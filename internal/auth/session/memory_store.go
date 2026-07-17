package session

import (
	"context"
	"sync"
	"time"
)

// MemoryStore is an in-memory implementation of SessionStore intended for
// tests, single-process deployments and development when Redis is not
// available. It is safe for concurrent use.
//
// The store keeps three indexes to satisfy the lookup contract without
// scanning:
//
//   - byAccess: access-token hash → session
//   - byRefresh: refresh-token hash → access-token hash
//   - byUser: user ID → set of access-token hashes
//   - byFamily: family ID → set of access-token hashes
type MemoryStore struct {
	mu        sync.RWMutex
	byAccess  map[string]*Session
	byRefresh map[string]string // refresh-hash → access-hash
	byUser    map[string]map[string]struct{}
	byFamily  map[string]map[string]struct{}
}

// NewMemoryStore returns an empty MemoryStore.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		byAccess:  make(map[string]*Session),
		byRefresh: make(map[string]string),
		byUser:    make(map[string]map[string]struct{}),
		byFamily:  make(map[string]map[string]struct{}),
	}
}

// Create persists a session, replacing any existing entry with the same
// access-token hash.
func (s *MemoryStore) Create(_ context.Context, sess *Session) error {
	if sess == nil {
		return ErrTokenNotFound
	}
	clone := *sess

	s.mu.Lock()
	defer s.mu.Unlock()

	// If an existing session shares the access hash, remove the stale
	// secondary indexes before overwriting.
	if old, ok := s.byAccess[clone.AccessTokenHash]; ok {
		s.removeLocked(old)
	}

	s.byAccess[clone.AccessTokenHash] = &clone
	if clone.RefreshTokenHash != "" {
		s.byRefresh[clone.RefreshTokenHash] = clone.AccessTokenHash
	}
	if clone.UserID != "" {
		if _, ok := s.byUser[clone.UserID]; !ok {
			s.byUser[clone.UserID] = make(map[string]struct{})
		}
		s.byUser[clone.UserID][clone.AccessTokenHash] = struct{}{}
	}
	if clone.FamilyID != "" {
		if _, ok := s.byFamily[clone.FamilyID]; !ok {
			s.byFamily[clone.FamilyID] = make(map[string]struct{})
		}
		s.byFamily[clone.FamilyID][clone.AccessTokenHash] = struct{}{}
	}
	return nil
}

// GetByAccess returns the session keyed by access token hash.
func (s *MemoryStore) GetByAccess(_ context.Context, accessHash string) (*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sess, ok := s.byAccess[accessHash]
	if !ok {
		return nil, ErrTokenNotFound
	}
	clone := *sess
	return &clone, nil
}

// GetByRefresh returns the session keyed by refresh token hash.
func (s *MemoryStore) GetByRefresh(_ context.Context, refreshHash string) (*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	accessHash, ok := s.byRefresh[refreshHash]
	if !ok {
		return nil, ErrTokenNotFound
	}
	sess, ok := s.byAccess[accessHash]
	if !ok {
		return nil, ErrTokenNotFound
	}
	clone := *sess
	return &clone, nil
}

// UpdateLastSeen updates the LastSeenAt timestamp on the session.
func (s *MemoryStore) UpdateLastSeen(_ context.Context, accessHash string, ts time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.byAccess[accessHash]
	if !ok {
		return nil
	}
	sess.LastSeenAt = ts
	return nil
}

// MarkRefreshRevoked marks the session as revoked and links the old refresh
// hash to the replacement (when provided). The replacement mapping is what
// the manager uses to detect reuse.
func (s *MemoryStore) MarkRefreshRevoked(_ context.Context, refreshHash, replacement string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	accessHash, ok := s.byRefresh[refreshHash]
	if !ok {
		return ErrTokenNotFound
	}
	sess, ok := s.byAccess[accessHash]
	if !ok {
		return ErrTokenNotFound
	}
	sess.Revoked = true
	// Rotate the refresh index to point at the new refresh hash so that
	// future lookups by the old hash still resolve (as revoked) and the
	// manager can detect reuse.
	if replacement != "" {
		delete(s.byRefresh, refreshHash)
		s.byRefresh[replacement] = accessHash
		sess.RefreshTokenHash = replacement
	}
	return nil
}

// MarkAccessRevoked flips the Revoked flag on the session keyed by the
// access hash. The session remains in the store so subsequent reads can
// surface the revoked status.
func (s *MemoryStore) MarkAccessRevoked(_ context.Context, accessHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.byAccess[accessHash]
	if !ok {
		return ErrTokenNotFound
	}
	sess.Revoked = true
	return nil
}

// DeleteByAccess removes the session entry entirely.
func (s *MemoryStore) DeleteByAccess(_ context.Context, accessHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.byAccess[accessHash]
	if !ok {
		return nil
	}
	s.removeLocked(sess)
	return nil
}

// DeleteByRefresh removes the session entry by refresh hash.
func (s *MemoryStore) DeleteByRefresh(_ context.Context, refreshHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	accessHash, ok := s.byRefresh[refreshHash]
	if !ok {
		return nil
	}
	if sess, ok := s.byAccess[accessHash]; ok {
		s.removeLocked(sess)
	}
	return nil
}

// DeleteByFamily removes every session belonging to the family.
func (s *MemoryStore) DeleteByFamily(_ context.Context, familyID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	members, ok := s.byFamily[familyID]
	if !ok {
		return nil
	}
	for accessHash := range members {
		if sess, ok := s.byAccess[accessHash]; ok {
			s.removeLocked(sess)
		}
	}
	return nil
}

// DeleteByUser removes every session belonging to the user.
func (s *MemoryStore) DeleteByUser(_ context.Context, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	members, ok := s.byUser[userID]
	if !ok {
		return nil
	}
	for accessHash := range members {
		if sess, ok := s.byAccess[accessHash]; ok {
			s.removeLocked(sess)
		}
	}
	return nil
}

// ListByUser returns all sessions for a user (active and revoked).
func (s *MemoryStore) ListByUser(_ context.Context, userID string) ([]*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	members, ok := s.byUser[userID]
	if !ok {
		return nil, nil
	}
	out := make([]*Session, 0, len(members))
	for accessHash := range members {
		if sess, ok := s.byAccess[accessHash]; ok {
			clone := *sess
			out = append(out, &clone)
		}
	}
	return out, nil
}

// Close is a no-op for the in-memory store.
func (s *MemoryStore) Close() error { return nil }

// removeLocked deletes the session from every secondary index. The caller
// must hold s.mu for writing.
func (s *MemoryStore) removeLocked(sess *Session) {
	if sess == nil {
		return
	}
	delete(s.byAccess, sess.AccessTokenHash)
	if sess.RefreshTokenHash != "" {
		delete(s.byRefresh, sess.RefreshTokenHash)
	}
	if set, ok := s.byUser[sess.UserID]; ok {
		delete(set, sess.AccessTokenHash)
		if len(set) == 0 {
			delete(s.byUser, sess.UserID)
		}
	}
	if set, ok := s.byFamily[sess.FamilyID]; ok {
		delete(set, sess.AccessTokenHash)
		if len(set) == 0 {
			delete(s.byFamily, sess.FamilyID)
		}
	}
}
