package storage

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"
)

// S3Store uses Amazon S3 REST API with AWS Signature Version 4.
type S3Store struct {
	bucket string
	region string
	akid   string
	secret string
	client *http.Client
}

// NewS3Store creates an S3Store. Credentials from AWS_ACCESS_KEY_ID / AWS_SECRET_ACCESS_KEY.
func NewS3Store(bucket, region string) (*S3Store, error) {
	if bucket == "" {
		return nil, fmt.Errorf("S3 bucket name required")
	}
	if region == "" {
		region = os.Getenv("AWS_REGION")
	}
	if region == "" {
		region = "us-east-1"
	}
	akid := os.Getenv("AWS_ACCESS_KEY_ID")
	secret := os.Getenv("AWS_SECRET_ACCESS_KEY")
	if akid == "" || secret == "" {
		return nil, fmt.Errorf("AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY required for S3")
	}
	return &S3Store{
		bucket: bucket,
		region: region,
		akid:   akid,
		secret: secret,
		client: &http.Client{Timeout: 60 * time.Second},
	}, nil
}

// NewS3StoreWithCreds creates an S3Store with explicit credentials (tests).
func NewS3StoreWithCreds(bucket, region, akid, secret string) (*S3Store, error) {
	if bucket == "" || akid == "" || secret == "" {
		return nil, fmt.Errorf("bucket and credentials required")
	}
	if region == "" {
		region = "us-east-1"
	}
	return &S3Store{
		bucket: bucket,
		region: region,
		akid:   akid,
		secret: secret,
		client: &http.Client{Timeout: 60 * time.Second},
	}, nil
}

func (s *S3Store) objectURL(key string) string {
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucket, s.region, escapePath(key))
}

func (s *S3Store) Upload(ctx context.Context, key string, data []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, s.objectURL(key), bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.ContentLength = int64(len(data))
	s.signV4(req, http.MethodPut, key, data, "")
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
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.objectURL(key), nil)
	if err != nil {
		return nil, err
	}
	s.signV4(req, http.MethodGet, key, nil, "")
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
		return nil, fmt.Errorf("S3 download failed (%d): %s", resp.StatusCode, string(body))
	}
	return io.ReadAll(resp.Body)
}

// List returns object keys with the given prefix (ListObjectsV2).
func (s *S3Store) List(ctx context.Context, prefix string) ([]string, error) {
	var keys []string
	continuation := ""
	for {
		q := url.Values{}
		q.Set("list-type", "2")
		if prefix != "" {
			q.Set("prefix", prefix)
		}
		if continuation != "" {
			q.Set("continuation-token", continuation)
		}
		u := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/?%s", s.bucket, s.region, q.Encode())
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return nil, err
		}
		s.signV4(req, http.MethodGet, "", nil, q.Encode())
		resp, err := s.client.Do(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode >= 400 {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("S3 list failed (%d): %s", resp.StatusCode, string(body))
		}
		var page struct {
			IsTruncated           bool   `xml:"IsTruncated"`
			NextContinuationToken string `xml:"NextContinuationToken"`
			Contents              []struct {
				Key string `xml:"Key"`
			} `xml:"Contents"`
		}
		if err := xml.NewDecoder(resp.Body).Decode(&page); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("S3 list decode: %w", err)
		}
		resp.Body.Close()
		for _, c := range page.Contents {
			keys = append(keys, c.Key)
		}
		if !page.IsTruncated {
			break
		}
		continuation = page.NextContinuationToken
	}
	return keys, nil
}

func (s *S3Store) Delete(ctx context.Context, key string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, s.objectURL(key), nil)
	if err != nil {
		return err
	}
	s.signV4(req, http.MethodDelete, key, nil, "")
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 && resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("S3 delete failed (%d): %s", resp.StatusCode, string(body))
	}
	return nil
}

func (s *S3Store) Exists(ctx context.Context, key string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, s.objectURL(key), nil)
	if err != nil {
		return false, err
	}
	s.signV4(req, http.MethodHead, key, nil, "")
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

func (s *S3Store) signV4(req *http.Request, method, objectKey string, body []byte, query string) {
	t := time.Now().UTC()
	datetime := t.Format("20060102T150405Z")
	date := t.Format("20060102")

	payloadHash := sha256Hex(nil)
	if len(body) > 0 {
		payloadHash = sha256Hex(body)
	}
	req.Header.Set("x-amz-date", datetime)
	req.Header.Set("x-amz-content-sha256", payloadHash)

	canonicalURI := "/"
	if objectKey != "" {
		canonicalURI = "/" + escapePath(objectKey)
	}
	canonicalHeaders, signedHeaders := canonicalSignedHeaders(req)

	canonicalReq := method + "\n" + canonicalURI + "\n" + query + "\n" +
		canonicalHeaders + "\n" + signedHeaders + "\n" + payloadHash
	scope := date + "/" + s.region + "/s3/aws4_request"
	stringToSign := "AWS4-HMAC-SHA256\n" + datetime + "\n" + scope + "\n" + sha256Hex([]byte(canonicalReq))
	sig := hex.EncodeToString(hmacSHA256(s.signingKey(date), []byte(stringToSign)))
	req.Header.Set("Authorization", fmt.Sprintf(
		"AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		s.akid, scope, signedHeaders, sig,
	))
}

func (s *S3Store) signingKey(date string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+s.secret), []byte(date))
	kRegion := hmacSHA256(kDate, []byte(s.region))
	kService := hmacSHA256(kRegion, []byte("s3"))
	return hmacSHA256(kService, []byte("aws4_request"))
}

func canonicalSignedHeaders(req *http.Request) (string, string) {
	hdrs := map[string]string{"host": req.URL.Host}
	for k, v := range req.Header {
		lk := strings.ToLower(k)
		if lk == "authorization" {
			continue
		}
		hdrs[lk] = strings.TrimSpace(strings.Join(v, ","))
	}
	keys := make([]string, 0, len(hdrs))
	for k := range hdrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var ch, sh strings.Builder
	for i, k := range keys {
		ch.WriteString(k)
		ch.WriteString(":")
		ch.WriteString(hdrs[k])
		ch.WriteString("\n")
		if i > 0 {
			sh.WriteString(";")
		}
		sh.WriteString(k)
	}
	return ch.String(), sh.String()
}

func hmacSHA256(key, data []byte) []byte {
	m := hmac.New(sha256.New, key)
	m.Write(data)
	return m.Sum(nil)
}

func sha256Hex(b []byte) string {
	if b == nil {
		b = []byte{}
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func escapePath(key string) string {
	parts := strings.Split(key, "/")
	for i, p := range parts {
		parts[i] = url.PathEscape(p)
	}
	return strings.Join(parts, "/")
}
