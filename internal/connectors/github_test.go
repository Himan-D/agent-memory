package connectors

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestVerifyGitHubSignature(t *testing.T) {
	payload := []byte("hello world")
	secret := "my-secret-key"

	// compute valid signature
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	validSignature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	tests := []struct {
		name      string
		payload   []byte
		signature string
		secret    string
		expected  bool
	}{
		{
			name:      "Valid signature",
			payload:   payload,
			signature: validSignature,
			secret:    secret,
			expected:  true,
		},
		{
			name:      "Invalid signature",
			payload:   payload,
			signature: "sha256=invalid12345",
			secret:    secret,
			expected:  false,
		},
		{
			name:      "Empty secret",
			payload:   payload,
			signature: validSignature,
			secret:    "",
			expected:  false,
		},
		{
			name:      "Wrong signature format",
			payload:   payload,
			signature: hex.EncodeToString(mac.Sum(nil)), // Missing sha256= prefix
			secret:    secret,
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := verifyGitHubSignature(tt.payload, tt.signature, tt.secret)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
