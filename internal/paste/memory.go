package paste

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/redis/go-redis/v9"
)

func (s *Service) ensureMemoryBudget(ctx context.Context) error {
	limitMB := s.cfg.MaxTotalMemoryMB
	if limitMB <= 0 {
		return nil
	}
	limitBytes := uint64(limitMB) * 1024 * 1024
	const maxEvictions = 256
	for i := 0; i < maxEvictions; i++ {
		used, err := redisUsedBytes(ctx, s.rdb)
		if err != nil {
			return err
		}
		if used < limitBytes {
			return nil
		}
		ok, err := evictOldestPaste(ctx, s.rdb)
		if err != nil {
			return err
		}
		if !ok {
			return ErrRedisMemoryPressure
		}
	}
	return ErrRedisMemoryPressure
}

func redisUsedBytes(ctx context.Context, rdb *redis.Client) (uint64, error) {
	info, err := rdb.Info(ctx, "memory").Result()
	if err != nil {
		return 0, err
	}
	for _, line := range strings.Split(info, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "used_memory:") {
			v := strings.TrimPrefix(line, "used_memory:")
			n, err := strconv.ParseUint(strings.TrimSpace(v), 10, 64)
			if err != nil {
				return 0, err
			}
			return n, nil
		}
	}
	return 0, errors.New("used_memory not found in INFO")
}
