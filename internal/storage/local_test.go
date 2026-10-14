package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestNewLocalStore_CreatesDir(t *testing.T) {
	tempDir := t.TempDir()
	baseDir := filepath.Join(tempDir, "newdir")

	// Ensure the directory doesn't exist initially
	if _, err := os.Stat(baseDir); !os.IsNotExist(err) {
		t.Fatalf("expected directory %s to not exist", baseDir)
	}

	store, err := NewLocalStore(baseDir)
	if err != nil {
		t.Fatalf("failed to create local store: %v", err)
	}
	if store == nil {
		t.Fatalf("store is nil")
	}

	// Verify directory was created
	info, err := os.Stat(baseDir)
	if err != nil {
		t.Fatalf("failed to stat created directory: %v", err)
	}
	if !info.IsDir() {
		t.Errorf("expected %s to be a directory", baseDir)
	}
}

func TestLocalStore(t *testing.T) {
	baseDir := t.TempDir()
	store, err := NewLocalStore(baseDir)
	if err != nil {
		t.Fatalf("failed to create local store: %v", err)
	}

	ctx := context.Background()
	key := "test/file.txt"
	data := []byte("hello world")

	// 1. Upload
	if err := store.Upload(ctx, key, data); err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	// 2. Exists
	exists, err := store.Exists(ctx, key)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if !exists {
		t.Errorf("expected key %q to exist", key)
	}

	// Exists for non-existent key
	exists, err = store.Exists(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("Exists failed for non-existent key: %v", err)
	}
	if exists {
		t.Errorf("expected nonexistent key to not exist")
	}

	// 3. Download
	downloaded, err := store.Download(ctx, key)
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}
	if string(downloaded) != string(data) {
		t.Errorf("downloaded data mismatch: got %q, want %q", downloaded, data)
	}

	// Download non-existent key
	_, err = store.Download(ctx, "nonexistent")
	if err == nil {
		t.Errorf("expected error downloading non-existent key")
	}

	// 4. List
	// Add another file in a different directory to test list prefix
	if err := store.Upload(ctx, "other/file2.txt", []byte("other")); err != nil {
		t.Fatalf("Upload failed: %v", err)
	}
	if err := store.Upload(ctx, "test/file3.txt", []byte("test3")); err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	keys, err := store.List(ctx, "test")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(keys) != 2 {
		t.Errorf("expected 2 keys, got %d", len(keys))
	}

	// keys should contain "test/file.txt" and "test/file3.txt"
	hasFile1 := false
	hasFile3 := false
	for _, k := range keys {
		if k == "test/file.txt" {
			hasFile1 = true
		} else if k == "test/file3.txt" {
			hasFile3 = true
		}
	}
	if !hasFile1 || !hasFile3 {
		t.Errorf("List returned unexpected keys: %v", keys)
	}

	// List all
	keys, err = store.List(ctx, "")
	if err != nil {
		t.Fatalf("List all failed: %v", err)
	}
	if len(keys) != 3 {
		t.Errorf("expected 3 keys total, got %d", len(keys))
	}

	// 5. Delete
	if err := store.Delete(ctx, key); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deletion
	exists, err = store.Exists(ctx, key)
	if err != nil {
		t.Fatalf("Exists after delete failed: %v", err)
	}
	if exists {
		t.Errorf("expected key %q to be deleted", key)
	}
}
