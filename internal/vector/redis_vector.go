package vector

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"agent-memory/internal/config"
	"agent-memory/internal/memory/types"
)

const redisVectorIndex = "memory_vector_idx"

type RedisVectorClient struct {
	rdb        *redis.Client
	cfg        config.AppConfig
	vectorSize int
}

func NewRedisVectorClient(redisURL string, vectorSize int) (*RedisVectorClient, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	rdb := redis.NewClient(opts)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	c := &RedisVectorClient{
		rdb:        rdb,
		vectorSize: vectorSize,
	}
	if err := c.ensureIndex(context.Background()); err != nil {
		return nil, fmt.Errorf("redis ensure index: %w", err)
	}
	return c, nil
}

func (c *RedisVectorClient) ensureIndex(ctx context.Context) error {
	res := c.rdb.Do(ctx, "FT.INFO", redisVectorIndex)
	if res.Err() == nil {
		return nil
	}
	createArgs := []interface{}{
		"FT.CREATE", redisVectorIndex,
		"ON", "HASH",
		"PREFIX", "1", "mem:",
		"SCHEMA",
		"text", "TEXT",
		"entity_id", "TAG",
		"tenant_id", "TAG",
		"created_at", "NUMERIC",
		"last_accessed", "NUMERIC",
		"embedding", "VECTOR", "FLAT",
		 strconv.Itoa(6 + c.vectorSize*4),
		"TYPE", "FLOAT32",
		"DIM", strconv.Itoa(c.vectorSize),
		"DISTANCE_METRIC", "COSINE",
	}
	res = c.rdb.Do(ctx, createArgs...)
	if res.Err() != nil && !strings.Contains(res.Err().Error(), "Index already exists") {
		return fmt.Errorf("create index: %w", res.Err())
	}
	return nil
}

func float32ToBytes(vec []float32) []byte {
	buf := make([]byte, len(vec)*4)
	for i, v := range vec {
		u := float32ToUint32(v)
		buf[i*4] = byte(u)
		buf[i*4+1] = byte(u >> 8)
		buf[i*4+2] = byte(u >> 16)
		buf[i*4+3] = byte(u >> 24)
	}
	return buf
}

func float32ToUint32(f float32) uint32 {
	return uint32(int32(f))
}

func (c *RedisVectorClient) StoreEmbedding(ctx context.Context, text string, id string, embedding []float32, metadata map[string]interface{}) (string, error) {
	docID := fmt.Sprintf("mem:%d:%s", time.Now().UnixNano(), id)
	fields := map[string]interface{}{
		"text":          text,
		"entity_id":     id,
		"embedding":     float32ToBytes(embedding),
		"created_at":    time.Now().Unix(),
		"last_accessed": time.Now().Unix(),
	}
	for k, v := range metadata {
		fields[k] = fmt.Sprintf("%v", v)
	}
	if err := c.rdb.HSet(ctx, docID, fields).Err(); err != nil {
		return "", fmt.Errorf("store embedding: %w", err)
	}
	return docID, nil
}

func (c *RedisVectorClient) Search(ctx context.Context, query []float32, limit int, threshold float32, filters map[string]interface{}) ([]types.MemoryResult, error) {
	return c.searchWithFilters(ctx, query, limit, threshold, filters)
}

func (c *RedisVectorClient) SearchWithTenant(ctx context.Context, query []float32, limit int, threshold float32, filters map[string]interface{}, tenantID string) ([]types.MemoryResult, error) {
	if tenantID != "" {
		if filters == nil {
			filters = make(map[string]interface{})
		}
		filters["tenant_id"] = tenantID
	}
	return c.searchWithFilters(ctx, query, limit, threshold, filters)
}

func (c *RedisVectorClient) searchWithFilters(ctx context.Context, query []float32, limit int, threshold float32, filters map[string]interface{}) ([]types.MemoryResult, error) {
	queryBlob := float32ToBytes(query)
	args := []interface{}{
		"FT.SEARCH", redisVectorIndex,
		"*=>[KNN " + strconv.Itoa(limit) + " @embedding $vec AS score]",
		"PARAMS", 2, "vec", queryBlob,
		"DIALECT", 2,
		"LIMIT", 0, strconv.Itoa(limit),
		"RETURN", 4, "text", "entity_id", "tenant_id", "score",
	}
	if len(filters) > 0 {
		var filterParts []string
		for k, v := range filters {
			filterParts = append(filterParts, fmt.Sprintf("@%s:{%v}", k, v))
		}
		args[2] = "(" + strings.Join(filterParts, " ") + ")=>[KNN " + strconv.Itoa(limit) + " @embedding $vec AS score]"
	}
	res, err := c.rdb.Do(ctx, args...).Result()
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	return c.parseSearchResult(res, threshold)
}

func (c *RedisVectorClient) parseSearchResult(res interface{}, threshold float32) ([]types.MemoryResult, error) {
	arr, ok := res.([]interface{})
	if !ok || len(arr) < 1 {
		return nil, nil
	}
	total, _ := arr[0].(int64)
	_ = total
	var results []types.MemoryResult
	for i := 1; i+1 < len(arr); i += 2 {
		key, _ := arr[i].(string)
		fieldsArr, ok := arr[i+1].([]interface{})
		if !ok {
			continue
		}
		props := map[string]interface{}{}
		for j := 0; j+1 < len(fieldsArr); j += 2 {
			fk, _ := fieldsArr[j].(string)
			props[fk] = fieldsArr[j+1]
		}
		var score float32
		if s, ok := props["score"].(string); ok {
			if f, err := strconv.ParseFloat(s, 32); err == nil {
				score = float32(f)
			}
		}
		if score < threshold {
			continue
		}
		text, _ := props["text"].(string)
		entityID, _ := props["entity_id"].(string)
		_ = key
		results = append(results, types.MemoryResult{
			Entity: types.Entity{
				ID:         entityID,
				Properties: props,
			},
			Score:  score,
			Text:   text,
			Source: "redis",
		})
	}
	return results, nil
}

func (c *RedisVectorClient) UpdateMemory(ctx context.Context, id string, text string, metadata map[string]interface{}) error {
	fields := map[string]interface{}{
		"text":          text,
		"last_accessed": time.Now().Unix(),
	}
	for k, v := range metadata {
		fields[k] = fmt.Sprintf("%v", v)
	}
	if err := c.rdb.HSet(ctx, id, fields).Err(); err != nil {
		return fmt.Errorf("update memory: %w", err)
	}
	return nil
}

func (c *RedisVectorClient) DeleteMemory(ctx context.Context, id string) error {
	if err := c.rdb.Del(ctx, id).Err(); err != nil {
		return fmt.Errorf("delete memory: %w", err)
	}
	return nil
}

func (c *RedisVectorClient) UpdateVector(ctx context.Context, id string, embedding []float32) error {
	if err := c.rdb.HSet(ctx, id, "embedding", float32ToBytes(embedding)).Err(); err != nil {
		return fmt.Errorf("update vector: %w", err)
	}
	return nil
}

func (c *RedisVectorClient) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

func (c *RedisVectorClient) Close() error {
	return c.rdb.Close()
}
