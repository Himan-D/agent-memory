package storage

import (
	"os"
	"testing"
)

func TestNewBlobStore(t *testing.T) {
	// Create a temporary directory for local store tests
	tempDir, err := os.MkdirTemp("", "store_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name      string
		provider  string
		dataDir   string
		gcsBucket string
		s3Bucket  string
		awsRegion string
		wantType  interface{}
		wantErr   bool
	}{
		{
			name:      "Valid GCS config",
			provider:  "gcs",
			gcsBucket: "my-gcs-bucket",
			wantType:  &GCSStore{},
			wantErr:   false,
		},
		{
			name:      "Missing GCS bucket",
			provider:  "gcs",
			gcsBucket: "",
			wantType:  nil,
			wantErr:   true,
		},
		{
			name:      "Valid S3 config",
			provider:  "s3",
			s3Bucket:  "my-s3-bucket",
			awsRegion: "us-west-2",
			wantType:  &S3Store{},
			wantErr:   false,
		},
		{
			name:      "Missing S3 bucket",
			provider:  "s3",
			s3Bucket:  "",
			awsRegion: "us-west-2",
			wantType:  nil,
			wantErr:   true,
		},
		{
			name:     "Valid Local config (explicit provider)",
			provider: "local",
			dataDir:  tempDir,
			wantType: &LocalStore{},
			wantErr:  false,
		},
		{
			name:     "Valid Local config (default fallback)",
			provider: "unknown-provider",
			dataDir:  tempDir,
			wantType: &LocalStore{},
			wantErr:  false,
		},
		{
			name:     "Valid Local config (empty provider)",
			provider: "",
			dataDir:  tempDir,
			wantType: &LocalStore{},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, err := NewBlobStore(tt.provider, tt.dataDir, tt.gcsBucket, tt.s3Bucket, tt.awsRegion)

			if (err != nil) != tt.wantErr {
				t.Errorf("NewBlobStore() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if store == nil {
					t.Fatalf("NewBlobStore() returned nil store but wantErr is false")
				}

				switch tt.wantType.(type) {
				case *GCSStore:
					if _, ok := store.(*GCSStore); !ok {
						t.Errorf("NewBlobStore() type = %T, want *GCSStore", store)
					}
				case *S3Store:
					if _, ok := store.(*S3Store); !ok {
						t.Errorf("NewBlobStore() type = %T, want *S3Store", store)
					}
				case *LocalStore:
					if _, ok := store.(*LocalStore); !ok {
						t.Errorf("NewBlobStore() type = %T, want *LocalStore", store)
					}
				default:
					t.Fatalf("Unknown expected type in test definition: %T", tt.wantType)
				}
			}
		})
	}
}
