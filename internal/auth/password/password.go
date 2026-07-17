// Package password provides password hashing and verification utilities.
//
// The implementation uses bcrypt with the library default cost (currently 10).
// Hashes are produced via Hash and verified via Verify. Both functions are
// constant-time where possible and rely on the underlying crypto library.
package password

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// ErrInvalidHash is returned when a stored hash cannot be parsed.
var ErrInvalidHash = errors.New("password: invalid hash")

// Hash returns a bcrypt hash of the given plaintext password.
// An error is returned if the input is empty or hashing fails.
func Hash(plaintext string) (string, error) {
	if plaintext == "" {
		return "", errors.New("password: plaintext is empty")
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(plaintext), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("password: hash: %w", err)
	}
	return string(hashed), nil
}

// Verify reports whether plaintext matches the stored bcrypt hash.
// It returns false (without an error) when the hash is malformed so callers
// can treat invalid credentials uniformly.
func Verify(plaintext, hashed string) (bool, error) {
	if plaintext == "" || hashed == "" {
		return false, nil
	}
	err := bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plaintext))
	if err == nil {
		return true, nil
	}
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return false, nil
	}
	return false, fmt.Errorf("password: verify: %w", err)
}

// MustHash is a convenience wrapper that panics on error. It is intended for
// tests and one-off bootstrap paths only.
func MustHash(plaintext string) string {
	h, err := Hash(plaintext)
	if err != nil {
		panic(err)
	}
	return h
}
