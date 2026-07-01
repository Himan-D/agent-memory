package session

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// refreshFamilyReuseError is returned when a refresh token is presented
// twice. The Manager interprets this as a possible theft and revokes the
// whole family.
type refreshFamilyReuseError struct {
	FamilyID string
	UserID   string
}

func (e *refreshFamilyReuseError) Error() string {
	return fmt.Sprintf("session: refresh token reuse detected for family %s", e.FamilyID)
}

// Refresh exchanges a refresh token for a brand-new access/refresh pair.
//
// Rotation rules (single-use, family-based theft detection):
//
//  1. The presented refresh token is hashed and looked up.
//  2. If it is unknown or revoked, the entire family is revoked
//     (possible theft) and ErrTokenRevoked is returned.
//  3. If it is expired, ErrTokenRevoked is returned without revoking
//     other sessions.
//  4. Otherwise a new pair is generated with the same family ID, the
//     old refresh hash is marked revoked and replaced, and the new pair
//     is returned.
func (m *manager) Refresh(ctx context.Context, refreshToken string) (*TokenPair, *Session, error) {
	refreshToken, err := NormalizeToken(refreshToken)
	if err != nil {
		return nil, nil, ErrTokenRevoked
	}
	hash := HashToken(refreshToken)

	sess, err := m.store.GetByRefresh(ctx, hash)
	if err != nil {
		if errors.Is(err, ErrTokenNotFound) {
			return nil, nil, ErrTokenRevoked
		}
		return nil, nil, err
	}
	if sess == nil {
		return nil, nil, ErrTokenRevoked
	}
	if sess.Revoked {
		// Refresh token already used → potential theft. Revoke the
		// whole family and every session for the user.
		_ = m.store.DeleteByFamily(ctx, sess.FamilyID)
		_ = m.store.DeleteByUser(ctx, sess.UserID)
		return nil, nil, &refreshFamilyReuseError{FamilyID: sess.FamilyID, UserID: sess.UserID}
	}
	if !sess.RefreshExpiresAt.IsZero() && m.cfg.now().After(sess.RefreshExpiresAt) {
		return nil, nil, ErrTokenRevoked
	}

	// Mint the replacement pair.
	newAccess, err := GenerateAccessToken()
	if err != nil {
		return nil, nil, fmt.Errorf("session: refresh generate access: %w", err)
	}
	newRefresh, err := GenerateRefreshToken()
	if err != nil {
		return nil, nil, fmt.Errorf("session: refresh generate refresh: %w", err)
	}

	now := m.cfg.now()
	newSess := &Session{
		UserID:           sess.UserID,
		Email:            sess.Email,
		Name:             sess.Name,
		Role:             sess.Role,
		AvatarURL:        sess.AvatarURL,
		FamilyID:         sess.FamilyID,
		AccessTokenHash:  HashToken(newAccess),
		RefreshTokenHash: HashToken(newRefresh),
		CreatedAt:        now,
		AccessExpiresAt:  now.Add(m.cfg.accessTTL()),
		RefreshExpiresAt: now.Add(m.cfg.refreshTTL()),
		LastSeenAt:       now,
		Revoked:          false,
	}

	// Persist the new session first so we never lose the old one if the
	// subsequent cleanup fails.
	if err := m.store.Create(ctx, newSess); err != nil {
		return nil, nil, fmt.Errorf("session: persist rotated session: %w", err)
	}

	// Mark the old refresh token as revoked and link it to the
	// replacement so reuse detection stays accurate.
	if err := m.store.MarkRefreshRevoked(ctx, hash, newSess.RefreshTokenHash); err != nil && !errors.Is(err, ErrTokenNotFound) {
		return nil, nil, fmt.Errorf("session: revoke old refresh: %w", err)
	}

	pair := &TokenPair{
		AccessToken:      newAccess,
		RefreshToken:     newRefresh,
		AccessExpiresAt:  newSess.AccessExpiresAt,
		RefreshExpiresAt: newSess.RefreshExpiresAt,
	}
	return pair, newSess, nil
}

// IsRefreshFamilyReuse reports whether the supplied error is a refresh
// token reuse error (i.e. theft signal).
func IsRefreshFamilyReuse(err error) bool {
	var fe *refreshFamilyReuseError
	return errors.As(err, &fe)
}

// NewUUID returns a UUID string. Wrapping uuid here keeps the rest of the
// package free of direct uuid imports and lets tests swap the generator.
func NewUUID() string {
	return uuid.NewString()
}

// TimeAfter reports whether t1 is strictly after t2. Helpers like this keep
// the rest of the file readable.
func TimeAfter(t1, t2 time.Time) bool {
	return t1.After(t2)
}
