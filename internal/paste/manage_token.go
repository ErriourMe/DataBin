package paste

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"

	"github.com/redis/go-redis/v9"
)

const manageTokenBytes = 32

var (
	ErrManageTokenRequired    = errors.New("token required")
	ErrInvalidManageToken     = errors.New("invalid token")
	ErrManageTokenUnavailable = errors.New("paste has no token")
)

func newManageToken() (plain string, hash string, err error) {
	buf := make([]byte, manageTokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}
	plain = base64.RawURLEncoding.EncodeToString(buf)
	return plain, hashManageToken(plain), nil
}

func hashManageToken(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}

func checkManageTokenMeta(ctx context.Context, rdb *redis.Client, metaKey, token string) error {
	stored, err := rdb.HGet(ctx, metaKey, "dhash").Result()
	if err == redis.Nil || stored == "" {
		return ErrManageTokenUnavailable
	}
	if err != nil {
		return err
	}
	if token == "" {
		return ErrManageTokenRequired
	}
	if subtle.ConstantTimeCompare([]byte(stored), []byte(hashManageToken(token))) != 1 {
		return ErrInvalidManageToken
	}
	return nil
}
