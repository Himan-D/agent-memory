// Package session implements custom email/password authentication for
// Hystersis. Sessions are issued as opaque access/refresh token pairs with
// single-use refresh tokens, family-based theft detection and configurable
// lifetimes.
//
// File layout:
//
//   - token.go     – token generation and hashing primitives.
//   - store.go     – SessionStore interface.
//   - manager.go   – SessionManager interface and core implementation.
//   - refresh.go   – refresh-token rotation + family revocation logic.
//   - memory_store.go / redis_store.go – concrete SessionStore implementations.
package session

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
)

// Default lifetimes used when callers do not supply explicit values.
const (
	DefaultAccessTTL  = 15 * 60              // 15 minutes
	DefaultRefreshTTL = 7 * 24 * 60 * 60     // 7 days
	GracePeriod       = 10                   // 10 seconds for token retry
	tokenBytes        = 32                   // 256-bit tokens
)

// ErrInvalidToken indicates that a token string is empty or malformed.
var ErrInvalidToken = errors.New("session: invalid token")

// ErrTokenNotFound indicates that the token is well-formed but not present
// in the store (expired, revoked, or never existed).
var ErrTokenNotFound = errors.New("session: token not found")

// ErrTokenRevoked indicates that the token was explicitly revoked.
var ErrTokenRevoked = errors.New("session: token revoked")

// GenerateAccessToken returns a URL-safe opaque token suitable for use as an
// access token. The token is 256 bits of entropy.
func GenerateAccessToken() (string, error) {
	return generateToken("access")
}

// GenerateRefreshToken returns a URL-safe opaque refresh token. The returned
// value is intended to be sent to the client exactly once; the server only
// stores its SHA-256 hash.
func GenerateRefreshToken() (string, error) {
	return generateToken("refresh")
}

// HashToken returns the SHA-256 hex digest of a token. Tokens are never
// stored in plaintext server-side; only their hashes are persisted.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func generateToken(_ string) (string, error) {
	buf := make([]byte, tokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("session: generate token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// NormalizeToken trims surrounding whitespace and rejects empty inputs.
func NormalizeToken(token string) (string, error) {
	if token == "" {
		return "", ErrInvalidToken
	}
	return token, nil
}
