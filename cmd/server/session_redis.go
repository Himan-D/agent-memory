package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisSessionStore is a Redis-backed session store with the same method signatures
// as SessionStore. Wire it via NewRedisSessionStore; fall back to NewSessionStore
// when Redis is not configured.
type RedisSessionStore struct {
	client *redis.Client
	ttl    time.Duration
}

// NewRedisSessionStore connects to Redis and returns a RedisSessionStore.
// Returns an error if the URL is invalid or Redis is unreachable.
func NewRedisSessionStore(redisURL string, ttl time.Duration) (*RedisSessionStore, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("invalid redis URL: %w", err)
	}
	client := redis.NewClient(opts)
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}
	return &RedisSessionStore{client: client, ttl: ttl}, nil
}

func (r *RedisSessionStore) sessionKey(token string) string {
	return "session:" + token
}

func (r *RedisSessionStore) userSessionsKey(userID string) string {
	return "user_sessions:" + userID
}

// CreateSession generates a token, stores session data in Redis with a TTL,
// and tracks the token in a per-user set. It revokes any prior session for the
// same user first (matching SessionStore behaviour).
func (r *RedisSessionStore) CreateSession(userID, email, name, role string) *Session {
	// Revoke existing sessions for this user.
	r.RevokeUserSessions(userID)

	token := generateSecureToken()
	now := time.Now()
	sess := &Session{
		Token:     token,
		UserID:    userID,
		Email:     email,
		Name:      name,
		Role:      role,
		CreatedAt: now,
		ExpiresAt: now.Add(r.ttl),
		LastSeen:  now,
	}

	ctx := context.Background()
	data, _ := json.Marshal(sess)
	r.client.Set(ctx, r.sessionKey(token), data, r.ttl)
	// Track token under this user so RevokeUserSessions can find it.
	r.client.SAdd(ctx, r.userSessionsKey(userID), token)
	r.client.Expire(ctx, r.userSessionsKey(userID), r.ttl)

	return sess
}

// ValidateToken retrieves and validates a session by token. Returns (nil, false)
// if not found or expired. Updates LastSeen on success.
func (r *RedisSessionStore) ValidateToken(token string) (*Session, bool) {
	ctx := context.Background()
	data, err := r.client.Get(ctx, r.sessionKey(token)).Bytes()
	if err != nil {
		return nil, false
	}

	var sess Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, false
	}

	if time.Now().After(sess.ExpiresAt) {
		r.client.Del(ctx, r.sessionKey(token))
		return nil, false
	}

	// Refresh LastSeen in-place (best-effort; don't fail on error).
	sess.LastSeen = time.Now()
	if updated, err := json.Marshal(&sess); err == nil {
		remaining := time.Until(sess.ExpiresAt)
		if remaining > 0 {
			r.client.Set(ctx, r.sessionKey(token), updated, remaining)
		}
	}

	return &sess, true
}

// RevokeSession deletes a single session token.
func (r *RedisSessionStore) RevokeSession(token string) {
	ctx := context.Background()
	data, err := r.client.Get(ctx, r.sessionKey(token)).Bytes()
	if err == nil {
		var sess Session
		if json.Unmarshal(data, &sess) == nil && sess.UserID != "" {
			r.client.SRem(ctx, r.userSessionsKey(sess.UserID), token)
		}
	}
	r.client.Del(ctx, r.sessionKey(token))
}

// RevokeUserSessions deletes all sessions belonging to a user.
func (r *RedisSessionStore) RevokeUserSessions(userID string) {
	ctx := context.Background()
	key := r.userSessionsKey(userID)
	tokens, err := r.client.SMembers(ctx, key).Result()
	if err != nil {
		return
	}
	for _, token := range tokens {
		r.client.Del(ctx, r.sessionKey(token))
	}
	r.client.Del(ctx, key)
}

// GetUserFromToken extracts user info from a valid token.
func (r *RedisSessionStore) GetUserFromToken(token string) (map[string]interface{}, bool) {
	sess, valid := r.ValidateToken(token)
	if !valid {
		return nil, false
	}
	return map[string]interface{}{
		"id":    sess.UserID,
		"email": sess.Email,
		"name":  sess.Name,
		"role":  sess.Role,
	}, true
}

// CleanupLoop is a no-op for Redis: TTL handles expiry automatically.
func (r *RedisSessionStore) CleanupLoop() {}
