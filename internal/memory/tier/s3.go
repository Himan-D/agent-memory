package tier

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
	"sort"
	"strings"
	"time"
)

type S3Archive struct {
	bucket string
	region string
	akid   string
	secret string
	client *http.Client
}

func NewS3Archive(bucket, region, akid, secret string) *S3Archive {
	return &S3Archive{
		bucket: bucket,
		region: region,
		akid:   akid,
		secret: secret,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *S3Archive) endpoint(key string) string {
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucket, s.region, url.PathEscape(key))
}

func (s *S3Archive) Write(ctx context.Context, memoryID string, data []byte) error {
	body := bytes.NewReader(data)
	req, err := http.NewRequestWithContext(ctx, "PUT", s.endpoint(memoryID), body)
	if err != nil {
		return fmt.Errorf("s3 write: %w", err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.ContentLength = int64(len(data))
	s.signV4(req, "PUT", memoryID, data)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("s3 write: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("s3 write: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return nil
}

func (s *S3Archive) Read(ctx context.Context, memoryID string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", s.endpoint(memoryID), nil)
	if err != nil {
		return nil, fmt.Errorf("s3 read: %w", err)
	}
	s.signV4(req, "GET", memoryID, nil)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("s3 read: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("s3 read %s: not found", memoryID)
	}
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("s3 read: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	return io.ReadAll(resp.Body)
}

func (s *S3Archive) Delete(ctx context.Context, memoryID string) error {
	req, err := http.NewRequestWithContext(ctx, "DELETE", s.endpoint(memoryID), nil)
	if err != nil {
		return fmt.Errorf("s3 delete: %w", err)
	}
	s.signV4(req, "DELETE", memoryID, nil)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("s3 delete: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 && resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("s3 delete: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return nil
}

func (s *S3Archive) List(ctx context.Context) ([]string, error) {
	var ids []string
	var marker string

	for {
		u := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/", s.bucket, s.region)
		if marker != "" {
			u += "?marker=" + url.QueryEscape(marker)
		}

		req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
		if err != nil {
			return nil, fmt.Errorf("s3 list: %w", err)
		}
		s.signV4(req, "GET", "", nil)

		resp, err := s.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("s3 list: %w", err)
		}

		if resp.StatusCode >= 300 {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("s3 list: %s: %s", resp.Status, strings.TrimSpace(string(body)))
		}

		var list struct {
			XMLName     xml.Name `xml:"ListBucketResult"`
			IsTruncated bool     `xml:"IsTruncated"`
			Contents    []struct {
				Key string `xml:"Key"`
			} `xml:"Contents"`
		}
		if err := xml.NewDecoder(resp.Body).Decode(&list); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("s3 list: decode: %w", err)
		}
		resp.Body.Close()

		for _, c := range list.Contents {
			ids = append(ids, c.Key)
		}

		if !list.IsTruncated || len(list.Contents) == 0 {
			break
		}
		marker = list.Contents[len(list.Contents)-1].Key
	}

	return ids, nil
}

func (s *S3Archive) signV4(req *http.Request, method, objectKey string, body []byte) {
	t := time.Now().UTC()
	datetime := t.Format("20060102T150405Z")
	date := t.Format("20060102")

	payloadHash := sha256Hex(nil)
	if len(body) > 0 {
		payloadHash = sha256Hex(body)
	}

	req.Header.Set("x-amz-date", datetime)
	req.Header.Set("x-amz-content-sha256", payloadHash)

	canonicalHeaders := s.buildCanonicalHeaders(req)
	signedHeaders := s.buildSignedHeaders(req)

	canonicalURI := "/" + url.PathEscape(objectKey)
	if objectKey == "" {
		canonicalURI = "/"
	}

	canonicalQuery := ""
	canonicalReq := method + "\n" + canonicalURI + "\n" + canonicalQuery + "\n" + canonicalHeaders + "\n" + signedHeaders + "\n" + payloadHash

	credentialScope := date + "/" + s.region + "/s3/aws4_request"
	stringToSign := "AWS4-HMAC-SHA256\n" + datetime + "\n" + credentialScope + "\n" + sha256Hex([]byte(canonicalReq))

	signingKey := s.buildSigningKey(date)
	signature := hex.EncodeToString(s.hmacSHA256(signingKey, []byte(stringToSign)))

	authHeader := fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		s.akid, credentialScope, signedHeaders, signature)
	req.Header.Set("Authorization", authHeader)
}

func (s *S3Archive) buildCanonicalHeaders(req *http.Request) string {
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

func (s *S3Archive) buildSignedHeaders(req *http.Request) string {
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

func (s *S3Archive) buildSigningKey(date string) []byte {
	kDate := s.hmacSHA256([]byte("AWS4"+s.secret), []byte(date))
	kRegion := s.hmacSHA256(kDate, []byte(s.region))
	kService := s.hmacSHA256(kRegion, []byte("s3"))
	return s.hmacSHA256(kService, []byte("aws4_request"))
}

func (s *S3Archive) hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
