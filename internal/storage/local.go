package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// LocalStore implements BlobStore using the local filesystem.
type LocalStore struct {
	baseDir string
}

// NewLocalStore creates a LocalStore rooted at baseDir, creating it if absent.
func NewLocalStore(baseDir string) (*LocalStore, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	return &LocalStore{baseDir: baseDir}, nil
}

func (s *LocalStore) Upload(_ context.Context, key string, data []byte) error {
	path := filepath.Join(s.baseDir, key)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (s *LocalStore) Download(_ context.Context, key string) ([]byte, error) {
	return os.ReadFile(filepath.Join(s.baseDir, key))
}

func (s *LocalStore) List(_ context.Context, prefix string) ([]string, error) {
	var keys []string
	root := filepath.Join(s.baseDir, prefix)
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(s.baseDir, path)
		keys = append(keys, rel)
		return nil
	})
	return keys, nil
}

func (s *LocalStore) Delete(_ context.Context, key string) error {
	return os.Remove(filepath.Join(s.baseDir, key))
}

func (s *LocalStore) Exists(_ context.Context, key string) (bool, error) {
	_, err := os.Stat(filepath.Join(s.baseDir, key))
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}
