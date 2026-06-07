package auth

import (
	"strings"
	"testing"
)

func TestHashAPIKeyUsesSalt(t *testing.T) {
	InitAPIKeySalt("test-salt")
	h1 := HashAPIKey("my-key")
	h2 := HashAPIKey("my-key")
	if h1 != h2 {
		t.Fatal("hash should be deterministic")
	}
	if len(h1) != 64 {
		t.Fatalf("expected sha256 hex length 64, got %d", len(h1))
	}

	InitAPIKeySalt("other-salt")
	h3 := HashAPIKey("my-key")
	if h1 == h3 {
		t.Fatal("different salts should produce different hashes")
	}
}

func TestGenerateTokenBundle(t *testing.T) {
	bundle, err := GenerateTokenBundle()
	if err != nil {
		t.Fatalf("GenerateTokenBundle: %v", err)
	}
	if len(bundle.APIKeySalt) != 64 {
		t.Fatalf("salt length: got %d", len(bundle.APIKeySalt))
	}
	if !strings.HasPrefix(bundle.AdminAPIKey, "am_admin_") {
		t.Fatalf("admin key prefix: %s", bundle.AdminAPIKey)
	}
	if !strings.HasPrefix(bundle.UserAPIKey, "usr_") {
		t.Fatalf("user key prefix: %s", bundle.UserAPIKey)
	}
	if bundle.SessionToken == "" || bundle.NextAuthSecret == "" || bundle.JWTSecret == "" {
		t.Fatal("expected non-empty secrets")
	}
}

func TestRandomSHA256(t *testing.T) {
	a, err := RandomSHA256()
	if err != nil {
		t.Fatal(err)
	}
	b, err := RandomSHA256()
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Fatal("expected unique random sha256 values")
	}
	if len(a) != 64 {
		t.Fatalf("sha256 hex length: %d", len(a))
	}
}
