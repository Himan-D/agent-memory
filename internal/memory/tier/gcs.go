package tier

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
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
				TokenURI    string `json:"token_uri"`
				ProjectID   string `json:"project_id"`
			}
			if json.Unmarshal(data, &sa) == nil && sa.ClientEmail != "" && sa.PrivateKey != "" {
				token, expiry, err := g.jwtBearerToken(ctx, sa.ClientEmail, sa.PrivateKey, sa.TokenURI)
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

func (g *GCSArchive) jwtBearerToken(ctx context.Context, clientEmail, privateKeyPEM, tokenURI string) (string, time.Time, error) {
	if tokenURI == "" {
		tokenURI = "https://oauth2.googleapis.com/token"
	}

	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return "", time.Time{}, fmt.Errorf("gcs: failed to decode PEM block from private key")
	}

	var key *rsa.PrivateKey
	switch block.Type {
	case "RSA PRIVATE KEY":
		var err error
		key, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return "", time.Time{}, fmt.Errorf("gcs: parse PKCS1 key: %w", err)
		}
	case "PRIVATE KEY":
		parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return "", time.Time{}, fmt.Errorf("gcs: parse PKCS8 key: %w", err)
		}
		var ok bool
		key, ok = parsed.(*rsa.PrivateKey)
		if !ok {
			return "", time.Time{}, fmt.Errorf("gcs: private key is not RSA")
		}
	default:
		return "", time.Time{}, fmt.Errorf("gcs: unsupported PEM block type: %s", block.Type)
	}

	now := time.Now().UTC()
	expiry := now.Add(1 * time.Hour)

	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))

	claimSet := fmt.Sprintf(
		`{"iss":"%s","scope":"https://www.googleapis.com/auth/devstorage.read_write","aud":"%s","iat":%d,"exp":%d}`,
		clientEmail, tokenURI, now.Unix(), expiry.Unix(),
	)
	payload := base64.RawURLEncoding.EncodeToString([]byte(claimSet))

	signingInput := header + "." + payload
	hashed := sha256.Sum256([]byte(signingInput))
	sigBytes, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, hashed[:])
	if err != nil {
		return "", time.Time{}, fmt.Errorf("gcs: sign JWT: %w", err)
	}
	signature := base64.RawURLEncoding.EncodeToString(sigBytes)

	jwt := signingInput + "." + signature

	form := url.Values{}
	form.Set("grant_type", "urn:ietf:params:oauth:grant-type:jwt-bearer")
	form.Set("assertion", jwt)

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURI, strings.NewReader(form.Encode()))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("gcs: token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := g.client.Do(req)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("gcs: token exchange: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("gcs: read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", time.Time{}, fmt.Errorf("gcs: token exchange failed (%d): %s", resp.StatusCode, body)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", time.Time{}, fmt.Errorf("gcs: decode token response: %w", err)
	}

	if tokenResp.ExpiresIn > 0 {
		expiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}

	return tokenResp.AccessToken, expiry, nil
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

	var ids []string
	pageToken := ""

	for {
		u := fmt.Sprintf("https://storage.googleapis.com/storage/v1/b/%s/o", g.bucket)
		if pageToken != "" {
			u += "?pageToken=" + url.QueryEscape(pageToken)
		}

		req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
		if err != nil {
			return nil, fmt.Errorf("gcs list: request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := g.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("gcs list: %w", err)
		}

		if resp.StatusCode >= 300 {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("gcs list: %s: %s", resp.Status, strings.TrimSpace(string(body)))
		}

		var result struct {
			Items       []struct {
				Name string `json:"name"`
			} `json:"items"`
			NextPageToken string `json:"nextPageToken"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("gcs list: decode: %w", err)
		}
		resp.Body.Close()

		for _, item := range result.Items {
			ids = append(ids, item.Name)
		}

		if result.NextPageToken == "" {
			break
		}
		pageToken = result.NextPageToken
	}

	return ids, nil
}
