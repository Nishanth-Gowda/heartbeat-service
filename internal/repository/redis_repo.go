package repository

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisRepository struct {
	rdb *redis.Client
}

func NewRedisRepository(rdb *redis.Client) *RedisRepository {
	return &RedisRepository{rdb: rdb}
}

func (r *RedisRepository) GetDeadServices(ctx context.Context, threshold int64) ([]int, error) {
	// Get all heartbeat keys
	keys, err := r.rdb.Keys(ctx, "heartbeat:*").Result()
	if err != nil {
		return nil, err
	}

	var dead_service_ids []int
	now := time.Now().Unix()

	for _, key := range keys {
		// Get the last heartbeat timestamp for this service
		last_heartbeat, err := r.rdb.Get(ctx, key).Int64()
		if err != nil {
			continue // Skip keys that don't have valid timestamps
		}

		// Check if heartbeat is older than threshold
		if now-last_heartbeat > threshold {
			// Extract service ID from key "heartbeat:123" -> 123
			parts := strings.Split(key, ":")
			if len(parts) == 2 {
				id, err := strconv.Atoi(parts[1])
				if err == nil {
					dead_service_ids = append(dead_service_ids, id)
				}
			}
		}
	}

	return dead_service_ids, nil
}

func (r *RedisRepository) RemoveService(ctx context.Context, service_id int) error {

	_, err := r.rdb.Del(ctx, fmt.Sprintf("heartbeat:%d", service_id)).Result()
	if err != nil {
		return err
	}

	return nil
}
