package sso

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFileStore_SaveLoadDelete(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sso-providers.json")
	store := NewFileStore(path)

	cfg := &Config{
		TenantID:     "tenant-1",
		ProviderType: ProviderTypeOAuth,
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		IssuerURL:    "https://issuer.example.com",
		CallbackURL:  "https://app.example.com/auth/sso/tenant-1/callback",
	}
	if err := store.Save(context.Background(), "tenant-1", cfg); err != nil {
		t.Fatalf("save: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("expected 0600 permissions, got %o", got)
	}

	configs, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(configs) != 1 {
		t.Fatalf("expected 1 config, got %d", len(configs))
	}
	if configs[0].TenantID != "tenant-1" || configs[0].ClientSecret != "client-secret" {
		t.Fatalf("unexpected config: %+v", configs[0])
	}

	updated := cloneConfig(cfg)
	updated.ClientSecret = "rotated"
	if err := store.Save(context.Background(), "tenant-1", updated); err != nil {
		t.Fatalf("save update: %v", err)
	}
	configs, err = store.Load(context.Background())
	if err != nil {
		t.Fatalf("load update: %v", err)
	}
	if len(configs) != 1 || configs[0].ClientSecret != "rotated" {
		t.Fatalf("expected updated config, got %+v", configs)
	}

	if err := store.Delete(context.Background(), "tenant-1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	configs, err = store.Load(context.Background())
	if err != nil {
		t.Fatalf("load after delete: %v", err)
	}
	if len(configs) != 0 {
		t.Fatalf("expected no configs after delete, got %d", len(configs))
	}
}

func TestFileStore_LoadMissingFile(t *testing.T) {
	store := NewFileStore(filepath.Join(t.TempDir(), "missing.json"))
	configs, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("load missing: %v", err)
	}
	if len(configs) != 0 {
		t.Fatalf("expected no configs, got %d", len(configs))
	}
}
