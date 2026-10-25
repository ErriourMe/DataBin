package paste

import (
	"context"
	"time"
)

const burnSeenKeyPrefix = "paste:burn_seen:"

func burnSeenKey(id string) string {
	return burnSeenKeyPrefix + id
}

func (s *Service) applyBurnSeenTTL(ctx context.Context, id string) error {
	if s.cfg.TTLHours <= 0 {
		return nil
	}
	ttl := time.Duration(s.cfg.TTLHours) * time.Hour
	return s.rdb.Expire(ctx, burnSeenKey(id), ttl).Err()
}
