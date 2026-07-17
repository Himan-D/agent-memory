package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// GCSStore uses Google Cloud Storage JSON REST API.
type GCSStore struct {
	bucket      string
	client      *http.Client
	token       string
	tokenExpiry time.Time
	mu          sync.Mutex
}

// NewGCSStore creates a GCS store. Auth via GOOGLE_ACCESS_TOKEN or
// GOOGLE_APPLICATION_CREDENTIALS (metadata fallback on GCE).
func NewGCSStore(bucket string) (*GCSStore, error) {
	if bucket == "" {
		return nil, fmt.Errorf("GCS bucket name required")
	}
	return &GCSStore{
		bucket: bucket,
		client: &http.Client{Timeout: 60 * time.Second},
	}, nil
}

func (s *GCSStore) auth(ctx context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.token != "" && time.Now().Before(s.tokenExpiry) {
		return s.token, nil
	}
	if t := os.Getenv("GOOGLE_ACCESS_TOKEN"); t != "" {
		s.token = t
		s.tokenExpiry = time.Now().Add(50 * time.Minute)
		return t, nil
	}
	// GCE metadata
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/token", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Metadata-Flavor", "Google")
	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("gcs auth: set GOOGLE_ACCESS_TOKEN or run on GCE (%w)", err)
	}
	defer resp.Body.Close()
	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.AccessToken == "" {
		return "", fmt.Errorf("gcs auth: empty token")
	}
	s.token = result.AccessToken
	s.tokenExpiry = time.Now().Add(time.Duration(result.ExpiresIn) * time.Second)
	return s.token, nil
}

func (s *GCSStore) Upload(ctx context.Context, key string, data []byte) error {
	token, err := s.auth(ctx)
	if err != nil {
		return err
	}
	u := fmt.Sprintf(
		"https://storage.googleapis.com/upload/storage/v1/b/%s/o?uploadType=media&name=%s",
		url.PathEscape(s.bucket), url.QueryEscape(key),
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
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
	token, err := s.auth(ctx)
	if err != nil {
		return nil, err
	}
	u := fmt.Sprintf(
		"https://storage.googleapis.com/storage/v1/b/%s/o/%s?alt=media",
		url.PathEscape(s.bucket), url.PathEscape(key),
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("object not found: %s", key)
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GCS download failed (%d): %s", resp.StatusCode, string(body))
	}
	return io.ReadAll(resp.Body)
}

// List returns object keys with the given prefix.
func (s *GCSStore) List(ctx context.Context, prefix string) ([]string, error) {
	token, err := s.auth(ctx)
	if err != nil {
		return nil, err
	}
	var keys []string
	pageToken := ""
	for {
		q := url.Values{}
		if prefix != "" {
			q.Set("prefix", prefix)
		}
		if pageToken != "" {
			q.Set("pageToken", pageToken)
		}
		u := fmt.Sprintf("https://storage.googleapis.com/storage/v1/b/%s/o?%s",
			url.PathEscape(s.bucket), q.Encode())
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := s.client.Do(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode >= 400 {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("GCS list failed (%d): %s", resp.StatusCode, string(body))
		}
		var page struct {
			NextPageToken string `json:"nextPageToken"`
			Items         []struct {
				Name string `json:"name"`
			} `json:"items"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body.Close()
		for _, it := range page.Items {
			if it.Name != "" {
				keys = append(keys, it.Name)
			}
		}
		if page.NextPageToken == "" {
			break
		}
		pageToken = page.NextPageToken
	}
	return keys, nil
}

func (s *GCSStore) Delete(ctx context.Context, key string) error {
	token, err := s.auth(ctx)
	if err != nil {
		return err
	}
	// GCS object names need URL-encoded path segments
	enc := strings.ReplaceAll(url.PathEscape(key), "+", "%20")
	u := fmt.Sprintf("https://storage.googleapis.com/storage/v1/b/%s/o/%s",
		url.PathEscape(s.bucket), enc)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode >= 400 && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("GCS delete failed (%d)", resp.StatusCode)
	}
	return nil
}

func (s *GCSStore) Exists(ctx context.Context, key string) (bool, error) {
	token, err := s.auth(ctx)
	if err != nil {
		return false, err
	}
	enc := strings.ReplaceAll(url.PathEscape(key), "+", "%20")
	u := fmt.Sprintf("https://storage.googleapis.com/storage/v1/b/%s/o/%s",
		url.PathEscape(s.bucket), enc)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := s.client.Do(req)
	if err != nil {
		return false, err
	}
	resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	return resp.StatusCode == http.StatusOK, nil
}
