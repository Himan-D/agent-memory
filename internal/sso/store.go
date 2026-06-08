package sso

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type Store interface {
	Load(ctx context.Context) ([]*Config, error)
	Save(ctx context.Context, tenantID string, cfg *Config) error
	Delete(ctx context.Context, tenantID string) error
}

type FileStore struct {
	path string
	mu   sync.Mutex
}

type persistedProviders struct {
	Providers []*Config `json:"providers"`
}

func NewFileStore(path string) *FileStore {
	return &FileStore{path: path}
}

func (s *FileStore) Load(ctx context.Context) ([]*Config, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.path == "" {
		return nil, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("sso store: read %s: %w", s.path, err)
	}
	if len(data) == 0 {
		return nil, nil
	}

	var persisted persistedProviders
	if err := json.Unmarshal(data, &persisted); err != nil {
		return nil, fmt.Errorf("sso store: parse %s: %w", s.path, err)
	}

	configs := make([]*Config, 0, len(persisted.Providers))
	for _, cfg := range persisted.Providers {
		if cfg != nil {
			configs = append(configs, cloneConfig(cfg))
		}
	}
	return configs, nil
}

func (s *FileStore) Save(ctx context.Context, tenantID string, cfg *Config) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.path == "" {
		return nil
	}
	if tenantID == "" {
		return fmt.Errorf("sso store: tenant ID is required")
	}
	if cfg == nil {
		return fmt.Errorf("sso store: config is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	persisted, err := s.loadLocked()
	if err != nil {
		return err
	}

	next := cloneConfig(cfg)
	next.TenantID = tenantID
	replaced := false
	for i, existing := range persisted.Providers {
		if existing != nil && existing.TenantID == tenantID {
			persisted.Providers[i] = next
			replaced = true
			break
		}
	}
	if !replaced {
		persisted.Providers = append(persisted.Providers, next)
	}

	return s.writeLocked(persisted)
}

func (s *FileStore) Delete(ctx context.Context, tenantID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.path == "" {
		return nil
	}
	if tenantID == "" {
		return fmt.Errorf("sso store: tenant ID is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	persisted, err := s.loadLocked()
	if err != nil {
		return err
	}

	providers := persisted.Providers[:0]
	for _, cfg := range persisted.Providers {
		if cfg != nil && cfg.TenantID != tenantID {
			providers = append(providers, cfg)
		}
	}
	persisted.Providers = providers
	return s.writeLocked(persisted)
}

func (s *FileStore) loadLocked() (*persistedProviders, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &persistedProviders{}, nil
		}
		return nil, fmt.Errorf("sso store: read %s: %w", s.path, err)
	}
	if len(data) == 0 {
		return &persistedProviders{}, nil
	}

	var persisted persistedProviders
	if err := json.Unmarshal(data, &persisted); err != nil {
		return nil, fmt.Errorf("sso store: parse %s: %w", s.path, err)
	}
	return &persisted, nil
}

func (s *FileStore) writeLocked(persisted *persistedProviders) error {
	if persisted == nil {
		persisted = &persistedProviders{}
	}
	data, err := json.MarshalIndent(persisted, "", "  ")
	if err != nil {
		return fmt.Errorf("sso store: marshal: %w", err)
	}
	data = append(data, '\n')

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("sso store: create dir %s: %w", dir, err)
	}

	tmp, err := os.CreateTemp(dir, ".sso-providers-*.json")
	if err != nil {
		return fmt.Errorf("sso store: create temp: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sso store: write temp: %w", err)
	}
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sso store: chmod temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("sso store: close temp: %w", err)
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		return fmt.Errorf("sso store: replace %s: %w", s.path, err)
	}
	return nil
}
