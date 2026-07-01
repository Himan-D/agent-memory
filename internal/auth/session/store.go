package session

import (
	"context"
	"time"
)

// Session is the server-side representation of an authenticated browser
// session. It holds the data needed to satisfy /auth/session, including
// only non-sensitive metadata. The plaintext access and refresh tokens are
// never persisted server-side; only their hashes are kept.
type Session struct {
	UserID           string    `json:"user_id"`
	Email            string    `json:"email"`
	Name             string    `json:"name"`
	Role             string    `json:"role"`
	AvatarURL        string    `json:"avatar_url,omitempty"`
	FamilyID         string    `json:"family_id"`
	AccessTokenHash  string    `json:"access_token_hash"`
	RefreshTokenHash string    `json:"refresh_token_hash"`
	CreatedAt        time.Time `json:"created_at"`
	AccessExpiresAt  time.Time `json:"access_expires_at"`
	RefreshExpiresAt time.Time `json:"refresh_expires_at"`
	LastSeenAt       time.Time `json:"last_seen_at"`
	Revoked          bool      `json:"revoked"`
}

// TokenPair is the value returned to clients when issuing new credentials.
type TokenPair struct {
	AccessToken      string    `json:"access_token"`
	RefreshToken     string    `json:"refresh_token"`
	AccessExpiresAt  time.Time `json:"access_expires_at"`
	RefreshExpiresAt time.Time `json:"refresh_expires_at"`
}

// SessionStore persists sessions. Implementations must be safe for
// concurrent use. Lookups are performed by token hash (not plaintext).
type SessionStore interface {
	// Create persists a brand-new session. The store must overwrite any
	// existing entry with the same access or refresh token hash.
	Create(ctx context.Context, s *Session) error

	// GetByAccess looks up a session by access token hash. It returns
	// ErrTokenNotFound if the token is unknown.
	GetByAccess(ctx context.Context, accessHash string) (*Session, error)

	// GetByRefresh looks up a session by refresh token hash.
	GetByRefresh(ctx context.Context, refreshHash string) (*Session, error)

	// UpdateLastSeen updates the LastSeenAt timestamp without touching
	// any other field. It is a no-op when the session is missing.
	UpdateLastSeen(ctx context.Context, accessHash string, ts time.Time) error

	// MarkRefreshRevoked flips the Revoked flag and replaces the stored
	// refresh hash with the supplied replacement (empty string when no
	// replacement exists yet).
	MarkRefreshRevoked(ctx context.Context, refreshHash, replacement string) error

	// MarkAccessRevoked flips the Revoked flag on the session keyed by
	// access hash.
	MarkAccessRevoked(ctx context.Context, accessHash string) error

	// DeleteByAccess removes the session entry. Idempotent.
	DeleteByAccess(ctx context.Context, accessHash string) error

	// DeleteByRefresh removes the session entry. Idempotent.
	DeleteByRefresh(ctx context.Context, refreshHash string) error

	// DeleteByFamily removes every session belonging to the given token
	// family. Used when theft is detected.
	DeleteByFamily(ctx context.Context, familyID string) error

	// DeleteByUser removes every session for the given user. Used when
	// the user's password changes or the account is disabled.
	DeleteByUser(ctx context.Context, userID string) error

	// ListByUser returns all active sessions for a user. The result is
	// intended for UI inspection and audit purposes.
	ListByUser(ctx context.Context, userID string) ([]*Session, error)

	// Close releases any underlying resources (e.g. Redis connection).
	Close() error
}
