package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"sync"
)

const legacyAPIKeySalt = "agent-memory-api-key-salt-v1"

var (
	saltMu  sync.RWMutex
	apiSalt string
)

// InitAPIKeySalt sets the salt used for API key hashing. Empty uses API_KEY_SALT
// env var, then falls back to the legacy default for backward compatibility.
func InitAPIKeySalt(salt string) {
	if salt == "" {
		salt = os.Getenv("API_KEY_SALT")
	}
	if salt == "" {
		salt = legacyAPIKeySalt
	}
	saltMu.Lock()
	apiSalt = salt
	saltMu.Unlock()
}

func currentSalt() string {
	saltMu.RLock()
	s := apiSalt
	saltMu.RUnlock()
	if s != "" {
		return s
	}
	if env := os.Getenv("API_KEY_SALT"); env != "" {
		return env
	}
	return legacyAPIKeySalt
}

// RandomHex returns cryptographically secure random bytes as lowercase hex.
func RandomHex(n int) (string, error) {
	if n < 0 {
		return "", fmt.Errorf("random hex: negative length %d", n)
	}
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("random hex: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// RandomSHA256 returns SHA256(random_bytes) as hex — a stable fingerprint for secrets.
func RandomSHA256() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("random sha256: %w", err)
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

// HashAPIKey hashes an API key with the configured salt using SHA-256.
func HashAPIKey(key string) string {
	h := sha256.New()
	h.Write([]byte(currentSalt()))
	h.Write([]byte(key))
	return hex.EncodeToString(h.Sum(nil))
}

// GenerateSessionToken returns a URL-safe base64 session token.
func GenerateSessionToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("session token: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// GenerateAdminAPIKey returns a new admin API key (am_admin_<sha256_hex>).
func GenerateAdminAPIKey() (string, error) {
	digest, err := RandomSHA256()
	if err != nil {
		return "", err
	}
	return "am_admin_" + digest, nil
}

// GenerateUserAPIKey returns a new tenant API key (usr_<sha256_hex>).
func GenerateUserAPIKey() (string, error) {
	digest, err := RandomSHA256()
	if err != nil {
		return "", err
	}
	return "usr_" + digest, nil
}

// GenerateSalt returns a new random API_KEY_SALT value (64 hex chars).
func GenerateSalt() (string, error) {
	return RandomHex(32)
}

// TokenBundle holds freshly generated credentials for bootstrap.
type TokenBundle struct {
	APIKeySalt    string `json:"api_key_salt"`
	AdminAPIKey   string `json:"admin_api_key"`
	UserAPIKey    string `json:"user_api_key"`
	SessionToken  string `json:"session_token"`
	NextAuthSecret string `json:"nextauth_secret"`
	JWTSecret     string `json:"jwt_secret"`
}

// GenerateTokenBundle creates a full set of bootstrap credentials.
func GenerateTokenBundle() (*TokenBundle, error) {
	salt, err := GenerateSalt()
	if err != nil {
		return nil, err
	}
	admin, err := GenerateAdminAPIKey()
	if err != nil {
		return nil, err
	}
	user, err := GenerateUserAPIKey()
	if err != nil {
		return nil, err
	}
	session, err := GenerateSessionToken()
	if err != nil {
		return nil, err
	}
	nextAuth, err := RandomHex(32)
	if err != nil {
		return nil, err
	}
	jwt, err := RandomHex(32)
	if err != nil {
		return nil, err
	}
	return &TokenBundle{
		APIKeySalt:     salt,
		AdminAPIKey:    admin,
		UserAPIKey:     user,
		SessionToken:   session,
		NextAuthSecret: nextAuth,
		JWTSecret:      jwt,
	}, nil
}
