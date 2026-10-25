package paste

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"databin/internal/config"
)

const (
	keyPrefix     = "paste:"
	metaKeyPrefix = "paste:meta:"

	// Default total opens for legacy burn pastes without burn_max in meta.
	burnAfterReadLegacyMaxViews = 2
	burnShowsMin                = 0
	burnShowsMax                = 99
)

var idPattern = regexp.MustCompile(`^[a-zA-Z0-9]{4,64}$`)

var (
	ErrNotFound         = errors.New("paste not found")
	ErrEditingDisabled  = errors.New("paste editing is disabled")
	ErrDeletionDisabled  = errors.New("paste deletion is disabled")
	ErrDeleteWindowClosed = errors.New("paste delete window has expired")
	ErrEditWindowClosed = errors.New("paste edit window has expired")
	ErrWrongPassword       = errors.New("incorrect password")
	ErrRedisMemoryPressure = errors.New("redis memory limit reached")
	ErrInvalidBurnShows    = errors.New("invalid burn view count")
)

type Info struct {
	Burn              bool
	BurnUnique        bool
	PasswordProtected bool
}

type Service struct {
	rdb *redis.Client
	cfg config.Paste
}

type StoreOptions struct {
	Burn       bool
	BurnShows  int // user setting N; paste is deleted after N+1 reads when Burn is true
	BurnUnique bool
	Password   string
}

type Created struct {
	ID    string
	Token string
}

// ReadOutcome is returned by Read. BurnViewsLeft is how many reads remain after this one (burn pastes only).
type ReadOutcome struct {
	Content       string
	BurnViewsLeft int
	BurnLastView  bool
}

func NewService(rdb *redis.Client, cfg config.Paste) *Service {
	return &Service{rdb: rdb, cfg: cfg}
}

func (s *Service) Store(ctx context.Context, content string, opts StoreOptions) (Created, error) {
	if len(content) > s.cfg.MaxBytes {
		return Created{}, fmt.Errorf("paste exceeds maximum size of %d bytes", s.cfg.MaxBytes)
	}
	phash, err := hashPassword(opts.Password)
	if err != nil {
		return Created{}, err
	}
	plainToken, dhash, err := newManageToken()
	if err != nil {
		return Created{}, err
	}
	if err := s.ensureMemoryBudget(ctx); err != nil {
		return Created{}, err
	}

	payload, err := encodeBody(content)
	if err != nil {
		return Created{}, err
	}

	const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	for attempt := 0; attempt < 8; attempt++ {
		id, err := randomKey(alphabet, s.cfg.KeyLength)
		if err != nil {
			return Created{}, err
		}
		dataKey := keyPrefix + id
		metaKey := metaKeyPrefix + id

		ok, err := s.rdb.SetNX(ctx, dataKey, payload, 0).Result()
		if err != nil {
			return Created{}, err
		}
		if !ok {
			continue
		}

		burn := "0"
		fields := []interface{}{"created", strconv.FormatInt(time.Now().Unix(), 10), "dhash", dhash}
		if opts.Burn {
			if opts.BurnShows < burnShowsMin || opts.BurnShows > burnShowsMax {
				_ = s.rdb.Del(ctx, dataKey).Err()
				return Created{}, ErrInvalidBurnShows
			}
			burn = "1"
			burnMax := opts.BurnShows + 1
			fields = append(fields, "burn", burn, "burn_max", strconv.Itoa(burnMax))
			if opts.BurnUnique {
				fields = append(fields, "burn_unique", "1")
			}
		} else {
			fields = append(fields, "burn", burn)
		}
		if phash != "" {
			fields = append(fields, "phash", phash)
		}
		if err := s.rdb.HSet(ctx, metaKey, fields...).Err(); err != nil {
			_ = s.rdb.Del(ctx, dataKey).Err()
			return Created{}, err
		}
		if err := s.applyTTL(ctx, dataKey, metaKey); err != nil {
			_ = s.rdb.Del(ctx, dataKey, metaKey).Err()
			return Created{}, err
		}
		if err := indexAdd(ctx, s.rdb, id, time.Now().Unix()); err != nil {
			_ = s.rdb.Del(ctx, dataKey, metaKey).Err()
			return Created{}, err
		}
		return Created{ID: id, Token: plainToken}, nil
	}
	return Created{}, errors.New("could not allocate paste id")
}

func (s *Service) Update(ctx context.Context, id, content, token string) error {
	if !idPattern.MatchString(id) {
		return errors.New("invalid paste id")
	}
	if !s.cfg.EditEnabled {
		return ErrEditingDisabled
	}
	if len(content) > s.cfg.MaxBytes {
		return fmt.Errorf("paste exceeds maximum size of %d bytes", s.cfg.MaxBytes)
	}

	dataKey := keyPrefix + id
	metaKey := metaKeyPrefix + id

	exists, err := s.rdb.Exists(ctx, dataKey).Result()
	if err != nil {
		return err
	}
	if exists == 0 {
		return ErrNotFound
	}

	if err := s.checkEditWindow(ctx, metaKey); err != nil {
		return err
	}
	if err := checkManageTokenMeta(ctx, s.rdb, metaKey, token); err != nil {
		return err
	}

	payload, err := encodeBody(content)
	if err != nil {
		return err
	}
	if err := s.rdb.Set(ctx, dataKey, payload, 0).Err(); err != nil {
		return err
	}
	return s.applyTTL(ctx, dataKey, metaKey)
}

// Read returns paste body. Burn-after-read pastes are deleted after burn_max reads.
// viewerKey identifies the client for burn_unique (empty skips unique tracking).
func (s *Service) Read(ctx context.Context, id, viewerKey string) (ReadOutcome, bool) {
	var out ReadOutcome
	if !idPattern.MatchString(id) {
		return out, false
	}
	dataKey := keyPrefix + id
	metaKey := metaKeyPrefix + id

	raw, err := s.rdb.Get(ctx, dataKey).Bytes()
	if err == redis.Nil {
		return out, false
	}
	if err != nil {
		return out, false
	}
	burn, _ := s.rdb.HGet(ctx, metaKey, "burn").Result()
	if burn == "1" {
		maxViews := burnAfterReadLegacyMaxViews
		if bm, err := s.rdb.HGet(ctx, metaKey, "burn_max").Result(); err == nil && bm != "" {
			if v, err := strconv.Atoi(bm); err == nil && v >= 1 {
				maxViews = v
			}
		}
		burnUnique := false
		if u, err := s.rdb.HGet(ctx, metaKey, "burn_unique").Result(); err == nil && u == "1" {
			burnUnique = true
		}
		countRead := true
		if burnUnique && viewerKey != "" {
			added, err := s.rdb.SAdd(ctx, burnSeenKey(id), viewerKey).Result()
			if err == nil {
				_ = s.applyBurnSeenTTL(ctx, id)
				if added == 0 {
					countRead = false
				}
			}
		}
		var n int64
		var burnCountErr error
		if countRead {
			n, burnCountErr = s.rdb.HIncrBy(ctx, metaKey, "burn_reads", 1).Result()
		} else {
			cur, burnCountErr := s.rdb.HGet(ctx, metaKey, "burn_reads").Result()
			if burnCountErr == redis.Nil {
				burnCountErr = nil
			} else if burnCountErr == nil && cur != "" {
				n, _ = strconv.ParseInt(cur, 10, 64)
			}
		}
		if burnCountErr == nil {
			remaining := maxViews - int(n)
			if remaining < 0 {
				remaining = 0
			}
			out.BurnViewsLeft = remaining
			if countRead && n >= int64(maxViews) {
				out.BurnLastView = true
				_ = s.deletePasteKeys(ctx, dataKey, metaKey)
			}
		}
	}
	plain, err := decodeBody(raw)
	if err != nil || plain == "" {
		return ReadOutcome{}, false
	}
	out.Content = plain
	return out, true
}

// Info for a paste without consuming it.
func (s *Service) Info(ctx context.Context, id string) (Info, bool) {
	if !idPattern.MatchString(id) {
		return Info{}, false
	}
	dataKey := keyPrefix + id
	metaKey := metaKeyPrefix + id
	exists, err := s.rdb.Exists(ctx, dataKey).Result()
	if err != nil || exists == 0 {
		return Info{}, false
	}
	burn := false
	if b, err := s.rdb.HGet(ctx, metaKey, "burn").Result(); err == nil && b == "1" {
		burn = true
	}
	burnUnique := false
	if u, err := s.rdb.HGet(ctx, metaKey, "burn_unique").Result(); err == nil && u == "1" {
		burnUnique = true
	}
	protected := false
	if ph, err := s.rdb.HGet(ctx, metaKey, "phash").Result(); err == nil && ph != "" {
		protected = true
	}
	return Info{Burn: burn, BurnUnique: burnUnique, PasswordProtected: protected}, true
}

func (s *Service) CheckPassword(ctx context.Context, id, password string) error {
	if !idPattern.MatchString(id) {
		return ErrNotFound
	}
	metaKey := metaKeyPrefix + id
	exists, err := s.rdb.Exists(ctx, keyPrefix+id).Result()
	if err != nil {
		return err
	}
	if exists == 0 {
		return ErrNotFound
	}
	return s.checkPasswordMeta(ctx, metaKey, password)
}

func (s *Service) checkPasswordMeta(ctx context.Context, metaKey, password string) error {
	phash, err := s.rdb.HGet(ctx, metaKey, "phash").Result()
	if err == redis.Nil || phash == "" {
		return nil
	}
	if err != nil {
		return err
	}
	if !checkPassword(phash, password) {
		return ErrWrongPassword
	}
	return nil
}

func (s *Service) Delete(ctx context.Context, id, token string) error {
	if !idPattern.MatchString(id) {
		return ErrNotFound
	}
	if !s.cfg.DeleteEnabled {
		return ErrDeletionDisabled
	}
	metaKey := metaKeyPrefix + id
	dataKey := keyPrefix + id
	exists, err := s.rdb.Exists(ctx, dataKey).Result()
	if err != nil {
		return err
	}
	if exists == 0 {
		return ErrNotFound
	}
	if err := s.checkDeleteWindow(ctx, metaKey); err != nil {
		return err
	}
	if err := checkManageTokenMeta(ctx, s.rdb, metaKey, token); err != nil {
		return err
	}
	return s.deletePasteKeys(ctx, dataKey, metaKey)
}

func (s *Service) checkEditWindow(ctx context.Context, metaKey string) error {
	return s.checkWindow(ctx, metaKey, s.cfg.EditMaxAgeSec, ErrEditWindowClosed)
}

func (s *Service) checkDeleteWindow(ctx context.Context, metaKey string) error {
	return s.checkWindow(ctx, metaKey, s.cfg.DeleteMaxAgeSec, ErrDeleteWindowClosed)
}

func (s *Service) checkWindow(ctx context.Context, metaKey string, maxSec int, expired error) error {
	if maxSec <= 0 {
		return nil
	}
	createdStr, err := s.rdb.HGet(ctx, metaKey, "created").Result()
	if err == redis.Nil {
		return nil
	}
	if err != nil {
		return err
	}
	created, err := strconv.ParseInt(createdStr, 10, 64)
	if err != nil {
		return nil
	}
	if time.Now().Unix()-created > int64(maxSec) {
		return expired
	}
	return nil
}

func (s *Service) applyTTL(ctx context.Context, dataKey, metaKey string) error {
	if s.cfg.TTLHours <= 0 {
		return nil
	}
	ttl := time.Duration(s.cfg.TTLHours) * time.Hour
	id := strings.TrimPrefix(dataKey, keyPrefix)
	pipe := s.rdb.Pipeline()
	pipe.Expire(ctx, dataKey, ttl)
	pipe.Expire(ctx, metaKey, ttl)
	pipe.Expire(ctx, statsKey(id), ttl)
	_, err := pipe.Exec(ctx)
	return err
}

func randomKey(alphabet string, length int) (string, error) {
	if length <= 0 {
		return "", errors.New("invalid key length")
	}
	max := big.NewInt(int64(len(alphabet)))
	buf := make([]byte, length)
	for i := range buf {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		buf[i] = alphabet[n.Int64()]
	}
	return string(buf), nil
}
