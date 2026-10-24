package services

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"time"

	"goravel/app/facades"
)

const pasteKeyPrefix = "paste:"

var pasteIDPattern = regexp.MustCompile(`^[a-zA-Z0-9]{4,64}$`)

type PasteService struct{}

func NewPasteService() *PasteService {
	return &PasteService{}
}

func (s *PasteService) Store(content string) (string, error) {
	maxBytes := facades.Config().GetInt("paste.max_bytes", 2*1024*1024)
	if len(content) > maxBytes {
		return "", fmt.Errorf("paste exceeds maximum size of %d bytes", maxBytes)
	}

	keyLength := facades.Config().GetInt("paste.key_length", 10)
	const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"

	for attempt := 0; attempt < 8; attempt++ {
		id, err := randomKey(alphabet, keyLength)
		if err != nil {
			return "", err
		}
		cacheKey := pasteKeyPrefix + id
		if facades.Cache().Has(cacheKey) {
			continue
		}

		ttlHours := facades.Config().GetInt("paste.ttl_hours", 0)
		var storeErr error
		if ttlHours > 0 {
			storeErr = facades.Cache().Put(cacheKey, content, time.Duration(ttlHours)*time.Hour)
		} else if !facades.Cache().Forever(cacheKey, content) {
			storeErr = errors.New("cache store failed")
		}
		if storeErr == nil {
			return id, nil
		}
	}

	return "", errors.New("could not allocate paste id")
}

func (s *PasteService) Update(id, content string) error {
	if !pasteIDPattern.MatchString(id) {
		return errors.New("invalid paste id")
	}
	maxBytes := facades.Config().GetInt("paste.max_bytes", 2*1024*1024)
	if len(content) > maxBytes {
		return fmt.Errorf("paste exceeds maximum size of %d bytes", maxBytes)
	}

	cacheKey := pasteKeyPrefix + id
	if !facades.Cache().Has(cacheKey) {
		return errors.New("paste not found")
	}

	ttlHours := facades.Config().GetInt("paste.ttl_hours", 0)
	if ttlHours > 0 {
		if err := facades.Cache().Put(cacheKey, content, time.Duration(ttlHours)*time.Hour); err != nil {
			return err
		}
	} else if !facades.Cache().Forever(cacheKey, content) {
		return errors.New("cache store failed")
	}
	return nil
}

func (s *PasteService) Get(id string) (string, bool) {
	if !pasteIDPattern.MatchString(id) {
		return "", false
	}
	cacheKey := pasteKeyPrefix + id
	if !facades.Cache().Has(cacheKey) {
		return "", false
	}
	return facades.Cache().GetString(cacheKey), true
}

func (s *PasteService) Delete(id string) bool {
	if !pasteIDPattern.MatchString(id) {
		return false
	}
	return facades.Cache().Forget(pasteKeyPrefix + id)
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
