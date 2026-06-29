package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStore is a Redis-backed implementation of the session Store interface.
// It is the hot working tier: low latency, time-bounded retention. For
// durable cold storage use Neo4jStore; combine via TieredStore.
//
// Key schema:
//   session:{user_id}:{session_id}:turns   -> JSON []QATurn          (TTL = cfg.TTL)
//   session:{user_id}:{session_id}:context -> JSON []ContextEntry    (TTL = cfg.TTL)
//   session:{user_id}:{session_id}:lessons -> JSON []DistilledLesson (TTL = cfg.TTL)
//
// All keys share the same TTL on write. There is no partial-key TTL.
type RedisStore struct {
	client    redisCmdable
	ttl       time.Duration
	keyPrefix string
}

// redisCmdable is the subset of *redis.Client used by RedisStore. It exists
// so tests can substitute an in-memory fake without spinning up Redis.
type redisCmdable interface {
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	Exists(ctx context.Context, keys ...string) *redis.IntCmd
	Ping(ctx context.Context) *redis.StatusCmd
}

// RedisStoreConfig configures a RedisStore.
type RedisStoreConfig struct {
	RedisURL  string
	TTL       time.Duration
	KeyPrefix string
}

// DefaultRedisStoreConfig returns production-ready defaults.
func DefaultRedisStoreConfig() RedisStoreConfig {
	return RedisStoreConfig{
		TTL:       168 * time.Hour, // 7 days
		KeyPrefix: "session:",
	}
}

// NewRedisStore connects to Redis and returns a RedisStore. Returns an
// error if the URL is invalid or the connection fails. Connection is
// validated with a 5s Ping.
func NewRedisStore(cfg RedisStoreConfig) (*RedisStore, error) {
	if cfg.TTL <= 0 {
		cfg.TTL = 168 * time.Hour
	}
	if cfg.KeyPrefix == "" {
		cfg.KeyPrefix = "session:"
	}

	client, err := newRedisClient(cfg.RedisURL)
	if err != nil {
		return nil, err
	}
	return &RedisStore{
		client:    client,
		ttl:       cfg.TTL,
		keyPrefix: cfg.KeyPrefix,
	}, nil
}

// newRedisClient builds a *redis.Client from a URL, matching the patterns
// used by internal/memory/tier/redis.go (ParseURL with fallback to bare
// address, PoolSize=10).
func newRedisClient(redisURL string) (*redis.Client, error) {
	if redisURL == "" {
		return nil, errors.New("redis URL is required")
	}
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		opts = &redis.Options{Addr: redisURL}
	}
	opts.PoolSize = 10

	client := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}
	return client, nil
}

// newRedisStoreFromClient is used by tests to inject a fake client.
func newRedisStoreFromClient(c redisCmdable, ttl time.Duration, prefix string) *RedisStore {
	if ttl <= 0 {
		ttl = 168 * time.Hour
	}
	if prefix == "" {
		prefix = "session:"
	}
	return &RedisStore{client: c, ttl: ttl, keyPrefix: prefix}
}

// Close releases the underlying connection pool. Safe to call on a nil
// receiver.
func (s *RedisStore) Close() error {
	if s == nil || s.client == nil {
		return nil
	}
	if c, ok := s.client.(*redis.Client); ok {
		return c.Close()
	}
	return nil
}

// Ping verifies the connection is healthy.
func (s *RedisStore) Ping(ctx context.Context) error {
	return s.client.Ping(ctx).Err()
}

// --- Store interface ---

func (s *RedisStore) AppendTurn(userID, sessionID string, turn QATurn) error {
	ctx := context.Background()
	key := s.turnsKey(userID, sessionID)
	turns, err := s.loadTurns(ctx, userID, sessionID)
	if err != nil {
		return err
	}
	turns = append(turns, turn)
	data, err := json.Marshal(turns)
	if err != nil {
		return fmt.Errorf("redis store: marshal turns: %w", err)
	}
	return s.client.Set(ctx, key, data, s.ttl).Err()
}

func (s *RedisStore) ListTurns(userID, sessionID string, lastN int) ([]QATurn, error) {
	turns, err := s.loadTurns(context.Background(), userID, sessionID)
	if err != nil {
		return nil, err
	}
	if lastN > 0 && len(turns) > lastN {
		turns = turns[len(turns)-lastN:]
	}
	return turns, nil
}

func (s *RedisStore) UpsertContext(userID string, entries []ContextEntry) error {
	ctx := context.Background()
	key := s.contextKey(userID, "")
	// Store under a single per-user key (no session id).
	data, err := json.Marshal(entries)
	if err != nil {
		return fmt.Errorf("redis store: marshal context: %w", err)
	}
	return s.client.Set(ctx, key, data, s.ttl).Err()
}

func (s *RedisStore) ListContext(userID string) ([]ContextEntry, error) {
	ctx := context.Background()
	key := s.contextKey(userID, "")
	val, err := s.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return []ContextEntry{}, nil
	}
	if err != nil {
		return nil, err
	}
	var entries []ContextEntry
	if err := json.Unmarshal([]byte(val), &entries); err != nil {
		return nil, fmt.Errorf("redis store: unmarshal context: %w", err)
	}
	return entries, nil
}

func (s *RedisStore) SaveLesson(userID string, lesson DistilledLesson) error {
	ctx := context.Background()
	// Lessons are keyed per session within the user namespace. If the
	// lesson has no session id, fall back to a per-user bucket.
	sessionBucket := lesson.SessionID
	key := s.lessonsKey(userID, sessionBucket)
	lessons, err := s.loadLessons(ctx, userID, sessionBucket)
	if err != nil {
		return err
	}
	lessons = append(lessons, lesson)
	data, err := json.Marshal(lessons)
	if err != nil {
		return fmt.Errorf("redis store: marshal lessons: %w", err)
	}
	return s.client.Set(ctx, key, data, s.ttl).Err()
}

func (s *RedisStore) ListLessons(userID string) ([]DistilledLesson, error) {
	// Iterate session buckets via SCAN. For correctness across deployments
	// we use SCAN on the prefix to collect all lesson keys, then load each.
	ctx := context.Background()
	pattern := s.keyPrefix + userID + ":*:lessons"
	c, ok := s.client.(redisScannable)
	if !ok {
		// Fallback: caller should use Neo4jStore for production scale.
		return nil, errors.New("redis store: client does not support SCAN")
	}
	var (
		cursor uint64
		out    []DistilledLesson
	)
	for {
		keys, next, err := c.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, err
		}
		for _, k := range keys {
			val, err := s.client.Get(ctx, k).Result()
			if errors.Is(err, redis.Nil) {
				continue
			}
			if err != nil {
				return nil, err
			}
			var lessons []DistilledLesson
			if err := json.Unmarshal([]byte(val), &lessons); err != nil {
				return nil, fmt.Errorf("redis store: unmarshal lessons key %s: %w", k, err)
			}
			out = append(out, lessons...)
		}
		if next == 0 {
			break
		}
		cursor = next
	}
	return out, nil
}

// --- helpers ---

func (s *RedisStore) turnsKey(userID, sessionID string) string {
	return s.keyPrefix + userID + ":" + sessionID + ":turns"
}

func (s *RedisStore) contextKey(userID, sessionID string) string {
	if sessionID == "" {
		return s.keyPrefix + userID + ":context"
	}
	return s.keyPrefix + userID + ":" + sessionID + ":context"
}

func (s *RedisStore) lessonsKey(userID, sessionID string) string {
	if sessionID == "" {
		return s.keyPrefix + userID + ":lessons"
	}
	return s.keyPrefix + userID + ":" + sessionID + ":lessons"
}

func (s *RedisStore) loadTurns(ctx context.Context, userID, sessionID string) ([]QATurn, error) {
	val, err := s.client.Get(ctx, s.turnsKey(userID, sessionID)).Result()
	if errors.Is(err, redis.Nil) {
		return []QATurn{}, nil
	}
	if err != nil {
		return nil, err
	}
	var turns []QATurn
	if err := json.Unmarshal([]byte(val), &turns); err != nil {
		return nil, fmt.Errorf("redis store: unmarshal turns: %w", err)
	}
	return turns, nil
}

func (s *RedisStore) loadLessons(ctx context.Context, userID, sessionID string) ([]DistilledLesson, error) {
	val, err := s.client.Get(ctx, s.lessonsKey(userID, sessionID)).Result()
	if errors.Is(err, redis.Nil) {
		return []DistilledLesson{}, nil
	}
	if err != nil {
		return nil, err
	}
	var lessons []DistilledLesson
	if err := json.Unmarshal([]byte(val), &lessons); err != nil {
		return nil, fmt.Errorf("redis store: unmarshal lessons: %w", err)
	}
	return lessons, nil
}

// redisScannable is the subset of *redis.Client that supports SCAN. Tests
// that don't need scan can use a fake that does not implement this.
type redisScannable interface {
	Scan(ctx context.Context, cursor uint64, match string, count int64) *redis.ScanCmd
}
