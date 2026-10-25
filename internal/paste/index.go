package paste

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

const pasteIndexKey = "paste:index"

func indexAdd(ctx context.Context, rdb *redis.Client, id string, createdUnix int64) error {
	return rdb.ZAdd(ctx, pasteIndexKey, redis.Z{
		Score:  float64(createdUnix),
		Member: id,
	}).Err()
}

func indexRemove(ctx context.Context, rdb *redis.Client, id string) error {
	return rdb.ZRem(ctx, pasteIndexKey, id).Err()
}

func evictOldestPaste(ctx context.Context, rdb *redis.Client) (bool, error) {
	ids, err := rdb.ZPopMin(ctx, pasteIndexKey, 1).Result()
	if err != nil {
		return false, err
	}
	if len(ids) == 0 {
		return false, nil
	}
	id := fmt.Sprint(ids[0].Member)
	if id == "" {
		return false, nil
	}
	dataKey := keyPrefix + id
	metaKey := metaKeyPrefix + id
	_ = rdb.Del(ctx, dataKey, metaKey, statsKey(id)).Err()
	return true, nil
}
