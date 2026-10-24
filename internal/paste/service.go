package paste

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"time"

	"github.com/redis/go-redis/v9"

	"databin/internal/config"
)

const keyPrefix = "paste:"

var idPattern = regexp.MustCompile(`^[a-zA-Z0-9]{4,64}$`)

type Service struct {
	rdb *redis.Client
	cfg config.Paste
}

func NewService(rdb *redis.Client, cfg config.Paste) *Service {
	return &Service{rdb: rdb, cfg: cfg}
}

func (s *Service) Store(ctx context.Context, content string) (string, error) {
	if len(content) > s.cfg.MaxBytes {
		return "", fmt.Errorf("paste exceeds maximum size of %d bytes", s.cfg.MaxBytes)
	}

	const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	for attempt := 0; attempt < 8; attempt++ {
		id, err := randomKey(alphabet, s.cfg.KeyLength)
		if err != nil {
			return "", err
		}
		cacheKey := keyPrefix + id
		exists, err := s.rdb.Exists(ctx, cacheKey).Result()
		if err != nil {
			return "", err
		}
		if exists > 0 {
			continue
		}
		if err := s.set(ctx, cacheKey, content); err != nil {
			return "", err
		}
		return id, nil
	}
	return "", errors.New("could not allocate paste id")
}

func (s *Service) Update(ctx context.Context, id, content string) error {
	if !idPattern.MatchString(id) {
		return errors.New("invalid paste id")
	}
	if len(content) > s.cfg.MaxBytes {
		return fmt.Errorf("paste exceeds maximum size of %d bytes", s.cfg.MaxBytes)
	}

	cacheKey := keyPrefix + id
	exists, err := s.rdb.Exists(ctx, cacheKey).Result()
	if err != nil {
		return err
	}
	if exists == 0 {
		return errors.New("paste not found")
	}
	return s.set(ctx, cacheKey, content)
}

func (s *Service) Get(ctx context.Context, id string) (string, bool) {
	if !idPattern.MatchString(id) {
		return "", false
	}
	val, err := s.rdb.Get(ctx, keyPrefix+id).Result()
	if err == redis.Nil {
		return "", false
	}
	if err != nil {
		return "", false
	}
	return val, true
}

func (s *Service) Delete(ctx context.Context, id string) bool {
	if !idPattern.MatchString(id) {
		return false
	}
	n, err := s.rdb.Del(ctx, keyPrefix+id).Result()
	return err == nil && n > 0
}

func (s *Service) set(ctx context.Context, key, content string) error {
	if s.cfg.TTLHours > 0 {
		return s.rdb.Set(ctx, key, content, time.Duration(s.cfg.TTLHours)*time.Hour).Err()
	}
	return s.rdb.Set(ctx, key, content, 0).Err()
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
