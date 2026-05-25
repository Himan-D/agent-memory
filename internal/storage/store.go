package storage

import (
	"context"
	"fmt"
)

// BlobStore provides a unified interface for object/blob storage across cloud providers.
type BlobStore interface {
	Upload(ctx context.Context, key string, data []byte) error
	Download(ctx context.Context, key string) ([]byte, error)
	List(ctx context.Context, prefix string) ([]string, error)
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
}

// NewBlobStore creates a BlobStore based on the provider config.
func NewBlobStore(provider, dataDir, gcsBucket, s3Bucket, awsRegion string) (BlobStore, error) {
	switch provider {
	case "gcs":
		if gcsBucket == "" {
			return nil, fmt.Errorf("GCS_BUCKET required when STORAGE_PROVIDER=gcs")
		}
		return NewGCSStore(gcsBucket)
	case "s3":
		if s3Bucket == "" {
			return nil, fmt.Errorf("S3_BUCKET required when STORAGE_PROVIDER=s3")
		}
		return NewS3Store(s3Bucket, awsRegion)
	default:
		return NewLocalStore(dataDir)
	}
}
