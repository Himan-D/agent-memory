package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	store, err := NewLocalStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := store.Upload(ctx, "a/b.txt", []byte("hello")); err != nil {
		t.Fatal(err)
	}
	data, err := store.Download(ctx, "a/b.txt")
	if err != nil || string(data) != "hello" {
		t.Fatalf("download: %s %v", data, err)
	}
	ok, err := store.Exists(ctx, "a/b.txt")
	if err != nil || !ok {
		t.Fatal("exists")
	}
	keys, err := store.List(ctx, "a")
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 1 {
		t.Fatalf("list: %v", keys)
	}
	if err := store.Delete(ctx, "a/b.txt"); err != nil {
		t.Fatal(err)
	}
	ok, _ = store.Exists(ctx, "a/b.txt")
	if ok {
		t.Fatal("should be gone")
	}
	// ensure path under base
	if _, err := os.Stat(filepath.Join(dir, "a")); err != nil && !os.IsNotExist(err) {
		t.Log(err)
	}
}

func TestNewBlobStoreProviders(t *testing.T) {
	s, err := NewBlobStore("local", t.TempDir(), "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if s == nil {
		t.Fatal("nil store")
	}
	// s3 without creds should fail
	if _, err := NewBlobStore("s3", "", "", "bucket", "us-east-1"); err == nil {
		t.Fatal("expected s3 creds error")
	}
}
