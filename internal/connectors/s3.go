package connectors

import (
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

type S3Client struct {
	region     string
	accessKey  string
	secretKey  string
	bucket     string
	endpoint   string
	httpClient *http.Client
}

type S3Object struct {
	Key          string `json:"key"`
	LastModified string `json:"last_modified"`
	ETag         string `json:"etag"`
	Size         int64  `json:"size"`
}

func NewS3Client(region, bucket, accessKey, secretKey, endpoint string) *S3Client {
	if endpoint == "" {
		endpoint = "https://" + bucket + ".s3." + region + ".amazonaws.com"
	}
	return &S3Client{
		region:     region,
		bucket:     bucket,
		accessKey:  accessKey,
		secretKey:  secretKey,
		endpoint:   endpoint,
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
	req.Header.Set("x-amz-content-sha256", sha256Hex(nil))
	c.signV4(req, "GET", "", nil)

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
	req.Header.Set("x-amz-content-sha256", sha256Hex(nil))
	c.signV4(req, "GET", key, nil)

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
	req.Header.Set("x-amz-content-sha256", sha256Hex(data))
	c.signV4(req, "PUT", key, data)

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

func (c *S3Client) signV4(req *http.Request, method, objectKey string, body []byte) {
	t := time.Now().UTC()
	datetime := t.Format("20060102T150405Z")
	date := t.Format("20060102")

	payloadHash := sha256Hex(nil)
	if len(body) > 0 {
		payloadHash = sha256Hex(body)
	}

	req.Header.Set("x-amz-date", datetime)
	req.Header.Set("x-amz-content-sha256", payloadHash)

	canonicalHeaders := buildCanonicalHeaders(req)
	signedHeaders := buildSignedHeaders(req)

	canonicalURI := "/" + url.PathEscape(objectKey)
	if objectKey == "" {
		canonicalURI = "/"
	}

	canonicalQuery := ""
	canonicalReq := method + "\n" + canonicalURI + "\n" + canonicalQuery + "\n" + canonicalHeaders + "\n" + signedHeaders + "\n" + payloadHash

	credentialScope := date + "/" + c.region + "/s3/aws4_request"
	stringToSign := "AWS4-HMAC-SHA256\n" + datetime + "\n" + credentialScope + "\n" + sha256Hex([]byte(canonicalReq))

	signingKey := buildSigningKey(c.secretKey, c.region, date)
	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

	authHeader := fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		c.accessKey, credentialScope, signedHeaders, signature)
	req.Header.Set("Authorization", authHeader)
}

func buildCanonicalHeaders(req *http.Request) string {
	hdrs := make(map[string]string)
	for k, v := range req.Header {
		lk := strings.ToLower(k)
		if lk == "authorization" {
			continue
		}
		hdrs[lk] = strings.TrimSpace(strings.Join(v, ","))
	}
	hdrs["host"] = req.URL.Host

	var keys []string
	for k := range hdrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteString(":")
		b.WriteString(hdrs[k])
		b.WriteString("\n")
	}
	return b.String()
}

func buildSignedHeaders(req *http.Request) string {
	var keys []string
	for k := range req.Header {
		lk := strings.ToLower(k)
		if lk == "authorization" {
			continue
		}
		keys = append(keys, lk)
	}
	keys = append(keys, "host")
	sort.Strings(keys)
	return strings.Join(keys, ";")
}

func buildSigningKey(secret, region, date string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secret), []byte(date))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte("s3"))
	return hmacSHA256(kService, []byte("aws4_request"))
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
