package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type EmbeddingCache struct {
	client *redis.Client
	ttl    time.Duration
}

type CacheEntry struct {
	Embedding []float32   `json:"embedding"`
	CreatedAt time.Time  `json:"created_at"`
	Text      string     `json:"text,omitempty"`
}

func NewEmbeddingCache(addr, password string, db int, ttl time.Duration) (*EmbeddingCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis connect: %w", err)
	}

	return &EmbeddingCache{
		client: client,
		ttl:    ttl,
	}, nil
}

func (c *EmbeddingCache) Get(ctx context.Context, key string) ([]float32, error) {
	data, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var entry CacheEntry
	if err := json.Unmarshal([]byte(data), &entry); err != nil {
		return nil, err
	}

	return entry.Embedding, nil
}

func (c *EmbeddingCache) Set(ctx context.Context, key string, embedding []float32) error {
	entry := CacheEntry{
		Embedding: embedding,
		CreatedAt: time.Now(),
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, key, data, c.ttl).Err()
}

func (c *EmbeddingCache) GetOrSet(ctx context.Context, key string, fetch func() ([]float32, error)) ([]float32, error) {
	emb, err := c.Get(ctx, key)
	if err == nil && emb != nil {
		return emb, nil
	}

	emb, err = fetch()
	if err != nil {
		return nil, err
	}

	if err := c.Set(ctx, key, emb); err != nil {
		return emb, nil
	}

	return emb, nil
}

func (c *EmbeddingCache) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

func (c *EmbeddingCache) Clear(ctx context.Context) error {
	return c.client.FlushDB(ctx).Err()
}

func (c *EmbeddingCache) Close() error {
	return c.client.Close()
}

func BuildCacheKey(text, model string) string {
	return fmt.Sprintf("embed:%s:%x", model, text)
}