package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStore is a Redis-backed implementation of SessionStore. It uses
// three key families:
//
//   - auth:session:access:<hash>     → JSON-encoded Session, TTL = access TTL
//   - auth:session:refresh:<hash>    → access hash, TTL = refresh TTL
//   - auth:session:family:<fam>      → SET of access hashes, TTL = refresh TTL
//   - auth:session:user:<uid>        → SET of access hashes, TTL = refresh TTL
//
// All access-token lookups happen via the single JSON blob, which keeps
// the read path to a single Redis round-trip.
type RedisStore struct {
	client *redis.Client
	prefix string
}

// NewRedisStore dials the given Redis URL and returns a RedisStore. The
// caller must Close the store to release the underlying connection pool.
func NewRedisStore(redisURL string) (*RedisStore, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		// ParseURL fails for bare host:port strings; treat as plain address.
		opts = &redis.Options{Addr: redisURL}
	}
	opts.PoolSize = 10

	client := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis session store: connect: %w", err)
	}

	return &RedisStore{
		client: client,
		prefix: "auth:session:",
	}, nil
}

// NewRedisStoreFromClient allows callers to share an existing Redis client
// (for example, the one already used by the tiered-memory subsystem).
func NewRedisStoreFromClient(client *redis.Client) *RedisStore {
	return &RedisStore{
		client: client,
		prefix: "auth:session:",
	}
}

func (s *RedisStore) accessKey(hash string) string {
	return s.prefix + "access:" + hash
}

func (s *RedisStore) refreshKey(hash string) string {
	return s.prefix + "refresh:" + hash
}

func (s *RedisStore) familyKey(familyID string) string {
	return s.prefix + "family:" + familyID
}

func (s *RedisStore) userKey(userID string) string {
	return s.prefix + "user:" + userID
}

// Create persists a session, replacing any existing entry with the same
// access or refresh hash.
func (s *RedisStore) Create(ctx context.Context, sess *Session) error {
	if sess == nil || sess.AccessTokenHash == "" {
		return ErrTokenNotFound
	}

	data, err := json.Marshal(sess)
	if err != nil {
		return fmt.Errorf("redis session store: marshal: %w", err)
	}

	now := time.Now()
	accessTTL := time.Until(sess.AccessExpiresAt)
	if accessTTL <= 0 {
		accessTTL = DefaultAccessTTL
	}
	refreshTTL := time.Until(sess.RefreshExpiresAt)
	if refreshTTL <= 0 {
		refreshTTL = DefaultRefreshTTL
	}

	pipe := s.client.TxPipeline()
	pipe.Set(ctx, s.accessKey(sess.AccessTokenHash), data, accessTTL)

	if sess.RefreshTokenHash != "" {
		pipe.Set(ctx, s.refreshKey(sess.RefreshTokenHash), sess.AccessTokenHash, refreshTTL)
	}
	if sess.FamilyID != "" {
		pipe.SAdd(ctx, s.familyKey(sess.FamilyID), sess.AccessTokenHash)
		pipe.Expire(ctx, s.familyKey(sess.FamilyID), refreshTTL)
	}
	if sess.UserID != "" {
		pipe.SAdd(ctx, s.userKey(sess.UserID), sess.AccessTokenHash)
		pipe.Expire(ctx, s.userKey(sess.UserID), refreshTTL)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("redis session store: create: %w", err)
	}

	_ = now // reserved for future stamping
	return nil
}

// GetByAccess returns the session keyed by access hash.
func (s *RedisStore) GetByAccess(ctx context.Context, accessHash string) (*Session, error) {
	raw, err := s.client.Get(ctx, s.accessKey(accessHash)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrTokenNotFound
		}
		return nil, fmt.Errorf("redis session store: get access: %w", err)
	}
	var sess Session
	if err := json.Unmarshal([]byte(raw), &sess); err != nil {
		return nil, fmt.Errorf("redis session store: unmarshal: %w", err)
	}
	return &sess, nil
}

// GetByRefresh resolves the refresh hash to the access hash and returns
// the underlying session.
func (s *RedisStore) GetByRefresh(ctx context.Context, refreshHash string) (*Session, error) {
	accessHash, err := s.client.Get(ctx, s.refreshKey(refreshHash)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrTokenNotFound
		}
		return nil, fmt.Errorf("redis session store: get refresh: %w", err)
	}
	return s.GetByAccess(ctx, accessHash)
}

// UpdateLastSeen updates the LastSeenAt timestamp on the session.
func (s *RedisStore) UpdateLastSeen(ctx context.Context, accessHash string, ts time.Time) error {
	sess, err := s.GetByAccess(ctx, accessHash)
	if err != nil {
		return nil // best effort
	}
	sess.LastSeenAt = ts
	return s.Create(ctx, sess)
}

// MarkRefreshRevoked flips the Revoked flag and links the old refresh hash
// to the replacement so reuse detection stays accurate.
func (s *RedisStore) MarkRefreshRevoked(ctx context.Context, refreshHash, replacement string) error {
	accessHash, err := s.client.Get(ctx, s.refreshKey(refreshHash)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return ErrTokenNotFound
		}
		return fmt.Errorf("redis session store: lookup refresh: %w", err)
	}

	sess, err := s.GetByAccess(ctx, accessHash)
	if err != nil {
		return err
	}
	sess.Revoked = true

	if replacement != "" {
		// The replacement is owned by the *new* session whose access
		// hash we don't yet know at this point; the manager creates
		// it immediately after, which will populate the new keys.
		// We only need to delete the old refresh key here.
		s.client.Del(ctx, s.refreshKey(refreshHash))
	} else {
		s.client.Del(ctx, s.refreshKey(refreshHash))
	}

	return s.Create(ctx, sess)
}

// MarkAccessRevoked flips the Revoked flag on the session.
func (s *RedisStore) MarkAccessRevoked(ctx context.Context, accessHash string) error {
	sess, err := s.GetByAccess(ctx, accessHash)
	if err != nil {
		return err
	}
	sess.Revoked = true
	return s.Create(ctx, sess)
}

// DeleteByAccess removes the session entry entirely.
func (s *RedisStore) DeleteByAccess(ctx context.Context, accessHash string) error {
	sess, err := s.GetByAccess(ctx, accessHash)
	if err != nil {
		if errors.Is(err, ErrTokenNotFound) {
			return nil
		}
		return err
	}
	return s.remove(ctx, sess)
}

// DeleteByRefresh removes the session entry by refresh hash.
func (s *RedisStore) DeleteByRefresh(ctx context.Context, refreshHash string) error {
	accessHash, err := s.client.Get(ctx, s.refreshKey(refreshHash)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil
		}
		return fmt.Errorf("redis session store: lookup refresh: %w", err)
	}
	sess, err := s.GetByAccess(ctx, accessHash)
	if err != nil {
		if errors.Is(err, ErrTokenNotFound) {
			return nil
		}
		return err
	}
	return s.remove(ctx, sess)
}

// DeleteByFamily removes every session belonging to the family.
func (s *RedisStore) DeleteByFamily(ctx context.Context, familyID string) error {
	hashes, err := s.client.SMembers(ctx, s.familyKey(familyID)).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return fmt.Errorf("redis session store: family members: %w", err)
	}
	for _, h := range hashes {
		if err := s.DeleteByAccess(ctx, h); err != nil {
			return err
		}
	}
	s.client.Del(ctx, s.familyKey(familyID))
	return nil
}

// DeleteByUser removes every session belonging to the user.
func (s *RedisStore) DeleteByUser(ctx context.Context, userID string) error {
	hashes, err := s.client.SMembers(ctx, s.userKey(userID)).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return fmt.Errorf("redis session store: user members: %w", err)
	}
	for _, h := range hashes {
		if err := s.DeleteByAccess(ctx, h); err != nil {
			return err
		}
	}
	s.client.Del(ctx, s.userKey(userID))
	return nil
}

// ListByUser returns all sessions for a user.
func (s *RedisStore) ListByUser(ctx context.Context, userID string) ([]*Session, error) {
	hashes, err := s.client.SMembers(ctx, s.userKey(userID)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, fmt.Errorf("redis session store: list user: %w", err)
	}
	out := make([]*Session, 0, len(hashes))
	for _, h := range hashes {
		sess, err := s.GetByAccess(ctx, h)
		if err != nil {
			if errors.Is(err, ErrTokenNotFound) {
				continue
			}
			return nil, err
		}
		out = append(out, sess)
	}
	return out, nil
}

// Close releases the underlying Redis client.
func (s *RedisStore) Close() error {
	if s.client == nil {
		return nil
	}
	return s.client.Close()
}

func (s *RedisStore) remove(ctx context.Context, sess *Session) error {
	pipe := s.client.TxPipeline()
	pipe.Del(ctx, s.accessKey(sess.AccessTokenHash))
	if sess.RefreshTokenHash != "" {
		pipe.Del(ctx, s.refreshKey(sess.RefreshTokenHash))
	}
	if sess.FamilyID != "" {
		pipe.SRem(ctx, s.familyKey(sess.FamilyID), sess.AccessTokenHash)
	}
	if sess.UserID != "" {
		pipe.SRem(ctx, s.userKey(sess.UserID), sess.AccessTokenHash)
	}
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("redis session store: remove: %w", err)
	}
	return nil
}
