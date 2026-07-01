package session

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// UserInfo captures the minimum identity data required to populate a
// Session. The Manager never reaches into the user store directly; callers
// pass this snapshot in. Keeping the boundary narrow lets the auth layer
// stay decoupled from internal/users.
type UserInfo struct {
	UserID    string
	Email     string
	Name      string
	Role      string
	AvatarURL string
}

// SessionManager issues and validates session token pairs. All operations
// are safe for concurrent use.
type SessionManager interface {
	// Create issues a brand-new access/refresh token pair for the user.
	// Any pre-existing sessions for the same user are kept; callers that
	// want single-session semantics should call RevokeAllForUser first.
	Create(ctx context.Context, user UserInfo) (*TokenPair, *Session, error)

	// ValidateAccess looks up a session by access token and returns it
	// when valid. The boolean is false when the token is unknown,
	// revoked or expired.
	ValidateAccess(ctx context.Context, accessToken string) (*Session, bool, error)

	// Refresh exchanges a refresh token for a brand-new pair. The old
	// refresh token is rotated (single-use). When the presented refresh
	// token has already been used, the entire family is revoked as a
	// theft-defence measure.
	Refresh(ctx context.Context, refreshToken string) (*TokenPair, *Session, error)

	// RevokeAccess deletes the session keyed by the given access token.
	// Idempotent; missing tokens are not an error.
	RevokeAccess(ctx context.Context, accessToken string) error

	// RevokeAllForUser removes every session belonging to the user.
	RevokeAllForUser(ctx context.Context, userID string) error
}

// ManagerConfig configures token lifetimes. Zero values fall back to the
// package defaults defined in token.go.
type ManagerConfig struct {
	AccessTTL  time.Duration
	RefreshTTL time.Duration
	// Now is overridable for tests; nil falls back to time.Now.
	Now func() time.Time
}

func (c ManagerConfig) accessTTL() time.Duration {
	if c.AccessTTL > 0 {
		return c.AccessTTL
	}
	return DefaultAccessTTL
}

func (c ManagerConfig) refreshTTL() time.Duration {
	if c.RefreshTTL > 0 {
		return c.RefreshTTL
	}
	return DefaultRefreshTTL
}

func (c ManagerConfig) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

// NewManager wires a SessionManager backed by the supplied store.
func NewManager(store SessionStore, cfg ManagerConfig) SessionManager {
	return &manager{store: store, cfg: cfg}
}

type manager struct {
	store SessionStore
	cfg   ManagerConfig
}

func (m *manager) Create(ctx context.Context, user UserInfo) (*TokenPair, *Session, error) {
	if user.UserID == "" {
		return nil, nil, errors.New("session: user.UserID is required")
	}

	accessToken, err := GenerateAccessToken()
	if err != nil {
		return nil, nil, fmt.Errorf("session: generate access token: %w", err)
	}
	refreshToken, err := GenerateRefreshToken()
	if err != nil {
		return nil, nil, fmt.Errorf("session: generate refresh token: %w", err)
	}

	now := m.cfg.now()
	sess := &Session{
		UserID:           user.UserID,
		Email:            user.Email,
		Name:             user.Name,
		Role:             user.Role,
		AvatarURL:        user.AvatarURL,
		FamilyID:         uuid.NewString(),
		AccessTokenHash:  HashToken(accessToken),
		RefreshTokenHash: HashToken(refreshToken),
		CreatedAt:        now,
		AccessExpiresAt:  now.Add(m.cfg.accessTTL()),
		RefreshExpiresAt: now.Add(m.cfg.refreshTTL()),
		LastSeenAt:       now,
		Revoked:          false,
	}

	if err := m.store.Create(ctx, sess); err != nil {
		return nil, nil, fmt.Errorf("session: persist: %w", err)
	}

	pair := &TokenPair{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		AccessExpiresAt:  sess.AccessExpiresAt,
		RefreshExpiresAt: sess.RefreshExpiresAt,
	}
	return pair, sess, nil
}

func (m *manager) ValidateAccess(ctx context.Context, accessToken string) (*Session, bool, error) {
	accessToken, err := NormalizeToken(accessToken)
	if err != nil {
		return nil, false, nil
	}
	hash := HashToken(accessToken)

	sess, err := m.store.GetByAccess(ctx, hash)
	if err != nil {
		if errors.Is(err, ErrTokenNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	if sess == nil {
		return nil, false, nil
	}
	if sess.Revoked {
		return nil, false, nil
	}
	if !sess.AccessExpiresAt.IsZero() && m.cfg.now().After(sess.AccessExpiresAt) {
		return nil, false, nil
	}

	// Best-effort last-seen update; do not fail validation on this.
	_ = m.store.UpdateLastSeen(ctx, hash, m.cfg.now())

	return sess, true, nil
}

func (m *manager) RevokeAccess(ctx context.Context, accessToken string) error {
	accessToken, err := NormalizeToken(accessToken)
	if err != nil {
		return nil
	}
	hash := HashToken(accessToken)
	if err := m.store.MarkAccessRevoked(ctx, hash); err != nil && !errors.Is(err, ErrTokenNotFound) {
		return err
	}
	return nil
}

func (m *manager) RevokeAllForUser(ctx context.Context, userID string) error {
	if userID == "" {
		return errors.New("session: userID is required")
	}
	return m.store.DeleteByUser(ctx, userID)
}
