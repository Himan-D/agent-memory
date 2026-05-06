package connectors

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type S3Client struct {
	region      string
	accessKey  string
	secretKey  string
	bucket     string
	endpoint   string
	httpClient *http.Client
}

type S3Object struct {
	Key          string `json:"key"`
	LastModified string `json:"last_modified"`
	ETag        string `json:"etag"`
	Size        int64  `json:"size"`
}

func NewS3Client(region, bucket, accessKey, secretKey, endpoint string) *S3Client {
	if endpoint == "" {
		endpoint = "https://" + bucket + ".s3." + region + ".amazonaws.com"
	}
	return &S3Client{
		region:    region,
		bucket:   bucket,
		accessKey: accessKey,
		secretKey: secretKey,
		endpoint: endpoint,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *S3Client) ListObjects(ctx context.Context, prefix string) ([]S3Object, error) {
	ep := c.endpoint
	if !strings.HasSuffix(ep, "/") {
		ep += "/"
	}
	if prefix != "" {
		ep += "?prefix=" + url.QueryEscape(prefix)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ep, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("x-amz-date", time.Now().UTC().Format("20060102T150405Z"))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("S3 error: %d", resp.StatusCode)
	}

	data, _ := io.ReadAll(resp.Body)
	_ = data
	var objects []S3Object

	return objects, nil
}

func (c *S3Client) GetObject(ctx context.Context, key string) ([]byte, error) {
	ep := c.endpoint
	if !strings.HasSuffix(ep, "/") {
		ep += "/"
	}
	ep += url.QueryEscape(key)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ep, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("S3 error: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func (c *S3Client) PutObject(ctx context.Context, key string, data []byte, contentType string) error {
	ep := c.endpoint
	if !strings.HasSuffix(ep, "/") {
		ep += "/"
	}
	ep += url.QueryEscape(key)

	body := strings.NewReader(string(data))
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, ep, body)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", contentType)
	req.Header.Set("x-amz-date", time.Now().UTC().Format("20060102T150405Z"))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("S3 error: %d", resp.StatusCode)
	}

	return nil
}