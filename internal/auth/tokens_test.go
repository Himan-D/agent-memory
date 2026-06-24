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

func TestRandomHex(t *testing.T) {
	tests := []struct {
		name    string
		n       int
		wantErr bool
		wantLen int
	}{
		{
			name:    "positive length",
			n:       16,
			wantErr: false,
			wantLen: 32, // hex encoding doubles the length
		},
		{
			name:    "zero length",
			n:       0,
			wantErr: false,
			wantLen: 0,
		},
		{
			name:    "negative length",
			n:       -1,
			wantErr: true,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RandomHex(tt.n)
			if (err != nil) != tt.wantErr {
				t.Errorf("RandomHex() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != tt.wantLen {
				t.Errorf("RandomHex() length = %v, want %v", len(got), tt.wantLen)
			}
		})
	}
}
