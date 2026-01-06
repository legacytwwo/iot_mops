package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type redisRepository struct {
	client *redis.Client
}

func New(client *redis.Client) *redisRepository {
	return &redisRepository{client: client}
}

func (s *redisRepository) SetLastSeen(ctx context.Context, deviceID string, ts time.Time) error {
	key := lastSeenKey(deviceID)
	if err := s.client.Set(ctx, key, ts.Unix(), 0).Err(); err != nil {
		return err
	}
	return nil
}

func lastSeenKey(deviceID string) string {
	return fmt.Sprintf("last_seen:%s", deviceID)
}
