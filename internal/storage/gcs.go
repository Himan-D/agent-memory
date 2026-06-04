package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// GCSStore uses Google Cloud Storage for blob storage via the JSON REST API.
// In production, replace the HTTP client with cloud.google.com/go/storage.
type GCSStore struct {
	bucket string
	client *http.Client
}

// NewGCSStore creates a GCSStore for the given bucket.
func NewGCSStore(bucket string) (*GCSStore, error) {
	if bucket == "" {
		return nil, fmt.Errorf("GCS bucket name required")
	}
	return &GCSStore{bucket: bucket, client: http.DefaultClient}, nil
}

func (s *GCSStore) Upload(ctx context.Context, key string, data []byte) error {
	url := fmt.Sprintf(
		"https://storage.googleapis.com/upload/storage/v1/b/%s/o?uploadType=media&name=%s",
		s.bucket, key,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GCS upload failed (%d): %s", resp.StatusCode, string(body))
	}
	return nil
}

func (s *GCSStore) Download(ctx context.Context, key string) ([]byte, error) {
	url := fmt.Sprintf(
		"https://storage.googleapis.com/storage/v1/b/%s/o/%s?alt=media",
		s.bucket, key,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("object not found: %s", key)
	}
	return io.ReadAll(resp.Body)
}

type gcsListResponse struct {
	Items []gcsObject `json:"items"`
}

type gcsObject struct {
	Name string `json:"name"`
}

func (s *GCSStore) List(ctx context.Context, prefix string) ([]string, error) {
	u := fmt.Sprintf(
		"https://storage.googleapis.com/storage/v1/b/%s/o?maxResults=1000",
		s.bucket,
	)
	if prefix != "" {
		u += "&prefix=" + url.QueryEscape(prefix)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GCS list failed (%d): %s", resp.StatusCode, string(body))
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var result gcsListResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parse GCS list response: %w", err)
	}
	names := make([]string, 0, len(result.Items))
	for _, obj := range result.Items {
		names = append(names, obj.Name)
	}
	return names, nil
}

func (s *GCSStore) Delete(ctx context.Context, key string) error {
	url := fmt.Sprintf(
		"https://storage.googleapis.com/storage/v1/b/%s/o/%s",
		s.bucket, key,
	)
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

func (s *GCSStore) Exists(ctx context.Context, key string) (bool, error) {
	url := fmt.Sprintf(
		"https://storage.googleapis.com/storage/v1/b/%s/o/%s",
		s.bucket, key,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
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
