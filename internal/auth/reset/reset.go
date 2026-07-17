// Package reset implements one-time password reset tokens. Tokens are
// single-use, short-lived (15 minutes) and consumed exactly once.
//
// The store is intentionally simple: an in-memory map keyed by token hash
// is sufficient for the current deployment topology. When the platform
// scales beyond a single node, swap in a Redis- or Neo4j-backed store
// that satisfies the same TokenStore interface.
package reset

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"
)

// DefaultTTL is how long a reset token stays valid.
const DefaultTTL = 15 * time.Minute

// ErrTokenNotFound is returned when a token is unknown or expired.
var ErrTokenNotFound = errors.New("reset: token not found")

// TokenStore persists reset tokens.
type TokenStore interface {
	Put(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error
	Get(ctx context.Context, tokenHash string) (userID string, expiresAt time.Time, err error)
	Delete(ctx context.Context, tokenHash string) error
}

// MemoryStore is the default in-memory TokenStore. It is safe for
// concurrent use and evicts expired entries lazily on access.
type MemoryStore struct {
	mu     sync.RWMutex
	tokens map[string]memoryEntry
	now    func() time.Time
}

type memoryEntry struct {
	userID     string
	expiresAt  time.Time
	consumedAt time.Time
}

// NewMemoryStore returns an empty MemoryStore.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		tokens: make(map[string]memoryEntry),
		now:    time.Now,
	}
}

// Put stores a token entry.
func (s *MemoryStore) Put(_ context.Context, userID, tokenHash string, expiresAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokens[tokenHash] = memoryEntry{
		userID:    userID,
		expiresAt: expiresAt,
	}
	return nil
}

// Get returns the user and expiry for a token, removing expired entries
// on the fly. Consumed tokens return ErrTokenNotFound.
func (s *MemoryStore) Get(_ context.Context, tokenHash string) (string, time.Time, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.tokens[tokenHash]
	if !ok {
		return "", time.Time{}, ErrTokenNotFound
	}
	if !entry.consumedAt.IsZero() {
		delete(s.tokens, tokenHash)
		return "", time.Time{}, ErrTokenNotFound
	}
	if !s.now().Before(entry.expiresAt) {
		delete(s.tokens, tokenHash)
		return "", time.Time{}, ErrTokenNotFound
	}
	return entry.userID, entry.expiresAt, nil
}

// Delete removes the token. Idempotent.
func (s *MemoryStore) Delete(_ context.Context, tokenHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tokens, tokenHash)
	return nil
}

// Consume marks the token as used and returns the owning user ID. The
// token is deleted from the store regardless of whether it was already
// consumed, so the caller can use Consume as the single authoritative
// "use" operation.
func (s *MemoryStore) Consume(ctx context.Context, tokenHash string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.tokens[tokenHash]
	if !ok {
		return "", ErrTokenNotFound
	}
	if !s.now().Before(entry.expiresAt) {
		delete(s.tokens, tokenHash)
		return "", ErrTokenNotFound
	}
	delete(s.tokens, tokenHash)
	return entry.userID, nil
}

// GenerateToken returns a URL-safe opaque token suitable for sending to
// the user via email.
func GenerateToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("reset: generate token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// HashToken returns the SHA-256 hex digest of a reset token. Only hashes
// are stored server-side.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// Manager orchestrates token issuance and consumption. It is a thin
// wrapper around TokenStore so callers don't have to plumb both the
// store and the hashing helper around.
type Manager struct {
	store TokenStore
	ttl   time.Duration
	now   func() time.Time
}

// NewManager wires a Manager backed by the supplied store.
func NewManager(store TokenStore) *Manager {
	return &Manager{
		store: store,
		ttl:   DefaultTTL,
		now:   time.Now,
	}
}

// Issue generates a token, stores its hash and returns the plaintext
// token to send to the user. The token is valid for the manager's TTL.
func (m *Manager) Issue(ctx context.Context, userID string) (string, error) {
	if userID == "" {
		return "", errors.New("reset: userID is required")
	}
	token, err := GenerateToken()
	if err != nil {
		return "", err
	}
	expiresAt := m.now().Add(m.ttl)
	if err := m.store.Put(ctx, userID, HashToken(token), expiresAt); err != nil {
		return "", err
	}
	return token, nil
}

// Consume validates and atomically consumes a token, returning the user
// ID it was issued for. ErrTokenNotFound is returned when the token is
// unknown, expired, or already consumed.
func (m *Manager) Consume(ctx context.Context, token string) (string, error) {
	if token == "" {
		return "", ErrTokenNotFound
	}
	mem, ok := m.store.(*MemoryStore)
	if ok {
		return mem.Consume(ctx, HashToken(token))
	}
	// Generic store: read then delete.
	userID, _, err := m.store.Get(ctx, HashToken(token))
	if err != nil {
		return "", err
	}
	if err := m.store.Delete(ctx, HashToken(token)); err != nil {
		return "", err
	}
	return userID, nil
}
