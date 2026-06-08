package sources

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

type R2BlobStore struct {
	accountID string
	bucket    string
	accessKey string
	secretKey string
	client    *http.Client
}

func NewR2BlobStore(accountID, bucket, accessKey, secretKey string) (*R2BlobStore, error) {
	if accountID == "" || bucket == "" || accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("R2_ACCOUNT_ID, R2_BUCKET, R2_ACCESS_KEY_ID, and R2_SECRET_ACCESS_KEY are required")
	}
	return &R2BlobStore{
		accountID: accountID,
		bucket:    bucket,
		accessKey: accessKey,
		secretKey: secretKey,
		client:    &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (r *R2BlobStore) Put(ctx context.Context, key string, data []byte, contentType string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, r.endpoint(key), bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("r2 put: %w", err)
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	req.Header.Set("Content-Type", contentType)
	req.ContentLength = int64(len(data))
	r.sign(req, http.MethodPut, key, data)

	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("r2 put: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("r2 put: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return nil
}

func (r *R2BlobStore) Delete(ctx context.Context, key string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, r.endpoint(key), nil)
	if err != nil {
		return fmt.Errorf("r2 delete: %w", err)
	}
	r.sign(req, http.MethodDelete, key, nil)

	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("r2 delete: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 && resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("r2 delete: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return nil
}

func (r *R2BlobStore) endpoint(key string) string {
	return fmt.Sprintf("https://%s.r2.cloudflarestorage.com/%s/%s", r.accountID, r.bucket, escapeObjectKey(key))
}

func (r *R2BlobStore) sign(req *http.Request, method, key string, body []byte) {
	t := time.Now().UTC()
	datetime := t.Format("20060102T150405Z")
	date := t.Format("20060102")

	payloadHash := sha256HexFull(body)
	req.Header.Set("x-amz-date", datetime)
	req.Header.Set("x-amz-content-sha256", payloadHash)

	canonicalHeaders := r.canonicalHeaders(req)
	signedHeaders := r.signedHeaders(req)
	canonicalURI := "/" + r.bucket
	if key != "" {
		canonicalURI += "/" + escapeObjectKey(key)
	}
	canonicalReq := method + "\n" + canonicalURI + "\n\n" + canonicalHeaders + "\n" + signedHeaders + "\n" + payloadHash

	scope := date + "/auto/s3/aws4_request"
	stringToSign := "AWS4-HMAC-SHA256\n" + datetime + "\n" + scope + "\n" + sha256HexFull([]byte(canonicalReq))
	signature := hex.EncodeToString(hmacSHA256(r.signingKey(date), []byte(stringToSign)))
	req.Header.Set("Authorization", fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s", r.accessKey, scope, signedHeaders, signature))
}

func (r *R2BlobStore) canonicalHeaders(req *http.Request) string {
	headers := map[string]string{"host": req.URL.Host}
	for k, v := range req.Header {
		lower := strings.ToLower(k)
		if lower == "authorization" {
			continue
		}
		headers[lower] = strings.TrimSpace(strings.Join(v, ","))
	}
	keys := make([]string, 0, len(headers))
	for k := range headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteByte(':')
		b.WriteString(headers[k])
		b.WriteByte('\n')
	}
	return b.String()
}

func (r *R2BlobStore) signedHeaders(req *http.Request) string {
	keys := []string{"host"}
	for k := range req.Header {
		lower := strings.ToLower(k)
		if lower != "authorization" {
			keys = append(keys, lower)
		}
	}
	sort.Strings(keys)
	return strings.Join(keys, ";")
}

func (r *R2BlobStore) signingKey(date string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+r.secretKey), []byte(date))
	kRegion := hmacSHA256(kDate, []byte("auto"))
	kService := hmacSHA256(kRegion, []byte("s3"))
	return hmacSHA256(kService, []byte("aws4_request"))
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	_, _ = h.Write(data)
	return h.Sum(nil)
}

func sha256HexFull(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func escapeObjectKey(key string) string {
	parts := strings.Split(strings.TrimLeft(key, "/"), "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}
