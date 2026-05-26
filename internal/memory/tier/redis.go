package tier

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisTierStore struct {
	client *redis.Client
	prefix string
}

func NewRedisTierStore(redisURL string) (*RedisTierStore, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     redisURL,
		PoolSize: 10,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	return &RedisTierStore{
		client: client,
		prefix: "tier:",
	}, nil
}

func (s *RedisTierStore) Get(ctx context.Context, key string) (string, error) {
	val, err := s.client.Get(ctx, s.prefix+key).Result()
	if err == redis.Nil {
		return "", nil
	}
	return val, err
}

func (s *RedisTierStore) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	return s.client.Set(ctx, s.prefix+key, value, ttl).Err()
}

func (s *RedisTierStore) Del(ctx context.Context, key string) error {
	return s.client.Del(ctx, s.prefix+key).Err()
}

func (s *RedisTierStore) Exists(ctx context.Context, key string) (bool, error) {
	n, err := s.client.Exists(ctx, s.prefix+key).Result()
	return n > 0, err
}

func (s *RedisTierStore) Close() error {
	return s.client.Close()
}
