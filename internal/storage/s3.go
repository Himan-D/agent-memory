package storage

import (
	"bytes"
	"context"
	"encoding/xml"
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

type s3ListBucketResult struct {
	XMLName xml.Name   `xml:"ListBucketResult"`
	Content []s3Object `xml:"Contents"`
}

type s3Object struct {
	Key string `xml:"Key"`
}

func (s *S3Store) List(ctx context.Context, prefix string) ([]string, error) {
	url := fmt.Sprintf("https://%s.s3.%s.amazonaws.com?list-type=2&prefix=%s", s.bucket, s.region, prefix)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
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
		return nil, fmt.Errorf("S3 list failed (%d): %s", resp.StatusCode, string(body))
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var result s3ListBucketResult
	if err := xml.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parse S3 list response: %w", err)
	}
	keys := make([]string, 0, len(result.Content))
	for _, obj := range result.Content {
		keys = append(keys, obj.Key)
	}
	return keys, nil
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
