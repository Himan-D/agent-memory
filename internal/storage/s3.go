package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
)

// S3Store uses Amazon S3 for blob storage via the REST API.
// In production, replace the HTTP client with github.com/aws/aws-sdk-go-v2.
type S3Store struct {
	bucket string
	region string
	client *http.Client
}

// NewS3Store creates an S3Store for the given bucket and region.
func NewS3Store(bucket, region string) (*S3Store, error) {
	if bucket == "" {
		return nil, fmt.Errorf("S3 bucket name required")
	}
	return &S3Store{bucket: bucket, region: region, client: http.DefaultClient}, nil
}

func (s *S3Store) Upload(ctx context.Context, key string, data []byte) error {
	url := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucket, s.region, key)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("S3 upload failed (%d): %s", resp.StatusCode, string(body))
	}
	return nil
}

func (s *S3Store) Download(ctx context.Context, key string) ([]byte, error) {
	url := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucket, s.region, key)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// List returns object keys with the given prefix.
// Simplified stub — returns empty; full impl requires XML parsing of the list response.
func (s *S3Store) List(_ context.Context, _ string) ([]string, error) {
	return []string{}, nil
}

func (s *S3Store) Delete(ctx context.Context, key string) error {
	url := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucket, s.region, key)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (s *S3Store) Exists(ctx context.Context, key string) (bool, error) {
	url := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucket, s.region, key)
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return false, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return false, err
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK, nil
}
