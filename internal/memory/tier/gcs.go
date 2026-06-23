package tier

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

type GCSArchive struct {
	bucket      string
	token       string
	tokenExpiry time.Time
	client      *http.Client
	mu          sync.Mutex
}

func NewGCSArchive(bucket string) *GCSArchive {
	return &GCSArchive{
		bucket: bucket,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (g *GCSArchive) getAccessToken(ctx context.Context) (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.token != "" && time.Now().Before(g.tokenExpiry) {
		return g.token, nil
	}

	credsFile := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if credsFile != "" {
		data, err := os.ReadFile(credsFile)
		if err == nil {
			var sa struct {
				ClientEmail string `json:"client_email"`
				PrivateKey  string `json:"private_key"`
			}
			if json.Unmarshal(data, &sa) == nil && sa.ClientEmail != "" && sa.PrivateKey != "" {
				token, expiry, err := g.jwtBearerToken(ctx, sa.ClientEmail, sa.PrivateKey)
				if err == nil {
					g.token = token
					g.tokenExpiry = expiry
					return token, nil
				}
			}
		}
	}

	req, err := http.NewRequestWithContext(ctx, "GET",
		"http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/token",
		nil,
	)
	if err != nil {
		return "", fmt.Errorf("gcs: metadata request: %w", err)
	}
	req.Header.Set("Metadata-Flavor", "Google")

	resp, err := g.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("gcs: metadata token: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("gcs: decode token: %w", err)
	}

	g.token = result.AccessToken
	g.tokenExpiry = time.Now().Add(time.Duration(result.ExpiresIn) * time.Second)
	return g.token, nil
}

func (g *GCSArchive) jwtBearerToken(ctx context.Context, clientEmail, privateKey string) (string, time.Time, error) {
	return "", time.Time{}, fmt.Errorf("gcs: JWT flow not implemented; use metadata server or ADC")
}

func (g *GCSArchive) Write(ctx context.Context, memoryID string, data []byte) error {
	token, err := g.getAccessToken(ctx)
	if err != nil {
		return fmt.Errorf("gcs write: %w", err)
	}

	u := fmt.Sprintf("https://storage.googleapis.com/upload/storage/v1/b/%s/o?uploadType=media&name=%s",
		g.bucket, url.PathEscape(memoryID))

	req, err := http.NewRequestWithContext(ctx, "POST", u, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("gcs write: request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("gcs write: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("gcs write: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return nil
}

func (g *GCSArchive) Read(ctx context.Context, memoryID string) ([]byte, error) {
	token, err := g.getAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("gcs read: %w", err)
	}

	u := fmt.Sprintf("https://storage.googleapis.com/storage/v1/b/%s/o/%s?alt=media",
		g.bucket, url.PathEscape(memoryID))

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, fmt.Errorf("gcs read: request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gcs read: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("gcs read %s: not found", memoryID)
	}
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gcs read: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	return io.ReadAll(resp.Body)
}

func (g *GCSArchive) Delete(ctx context.Context, memoryID string) error {
	token, err := g.getAccessToken(ctx)
	if err != nil {
		return fmt.Errorf("gcs delete: %w", err)
	}

	u := fmt.Sprintf("https://storage.googleapis.com/storage/v1/b/%s/o/%s",
		g.bucket, url.PathEscape(memoryID))

	req, err := http.NewRequestWithContext(ctx, "DELETE", u, nil)
	if err != nil {
		return fmt.Errorf("gcs delete: request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("gcs delete: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 && resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("gcs delete: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return nil
}

func (g *GCSArchive) List(ctx context.Context) ([]string, error) {
	token, err := g.getAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("gcs list: %w", err)
	}

	u := fmt.Sprintf("https://storage.googleapis.com/storage/v1/b/%s/o", g.bucket)

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, fmt.Errorf("gcs list: request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gcs list: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gcs list: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var result struct {
		Items []struct {
			Name string `json:"name"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("gcs list: decode: %w", err)
	}

	var ids []string
	for _, item := range result.Items {
		ids = append(ids, item.Name)
	}
	return ids, nil
}
