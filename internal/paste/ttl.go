package paste

import (
	"context"
	"time"
)

// RemainingTTL returns seconds until paste expiry, or -1 if no expiry.
func (s *Service) RemainingTTL(ctx context.Context, id string) (int, bool) {
	if !idPattern.MatchString(id) {
		return 0, false
	}
	ttl, err := s.rdb.TTL(ctx, keyPrefix+id).Result()
	if err != nil || ttl < 0 {
		if ttl == -2 {
			return 0, false
		}
		return -1, true
	}
	return int(ttl / time.Second), true
}

func (s *Service) CacheableRaw(ctx context.Context, id string) (maxAgeSec int, ok bool) {
	info, ok := s.Info(ctx, id)
	if !ok {
		return 0, false
	}
	if info.Burn {
		return 0, false
	}
	if s.cfg.EditEnabled {
		return 0, false
	}
	remaining, exists := s.RemainingTTL(ctx, id)
	if !exists {
		return 0, false
	}
	if remaining > 0 {
		return remaining, true
	}
	if remaining == -1 && s.cfg.TTLHours <= 0 {
		return 31536000, true
	}
	if remaining == -1 && s.cfg.TTLHours > 0 {
		return int(time.Duration(s.cfg.TTLHours) * time.Hour / time.Second), true
	}
	return 0, false
}
