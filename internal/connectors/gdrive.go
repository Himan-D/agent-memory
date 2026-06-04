package connectors

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type GoogleDriveClient struct {
	clientID     string
	clientSecret string
	accessToken  string
	refreshToken string
	httpClient   *http.Client
}

type GFile struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	MimeType string            `json:"mimeType"`
	Link     string            `json:"link"`
	Size     int64             `json:"size,omitempty"`
	Modified time.Time         `json:"modified,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type driveFile struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	MimeType string `json:"mimeType"`
	WebViewLink string `json:"webViewLink"`
	Size     string `json:"size"`
	ModifiedTime string `json:"modifiedTime"`
}

type driveFileList struct {
	Files          []driveFile `json:"files"`
	NextPageToken  string      `json:"nextPageToken"`
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

func NewGoogleDriveClient(accessToken, refreshToken, clientID, clientSecret string) *GoogleDriveClient {
	return &GoogleDriveClient{
		clientID:     clientID,
		clientSecret: clientSecret,
		accessToken:  accessToken,
		refreshToken: refreshToken,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *GoogleDriveClient) ListFiles(ctx context.Context, query string, limit int) ([]GFile, error) {
	if limit <= 0 {
		limit = 100
	}

	params := url.Values{}
	params.Set("pageSize", strconv.Itoa(limit))
	params.Set("fields", "files(id,name,mimeType,webViewLink,size,modifiedTime)")
	if query != "" {
		params.Set("q", query)
	}

	reqURL := "https://www.googleapis.com/drive/v3/files?" + params.Encode()
	body, err := c.doRequest(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("gdrive: list files: %w", err)
	}

	var fileList driveFileList
	if err := json.Unmarshal(body, &fileList); err != nil {
		return nil, fmt.Errorf("gdrive: unmarshal file list: %w", err)
	}

	files := make([]GFile, 0, len(fileList.Files))
	for _, f := range fileList.Files {
		size, _ := strconv.ParseInt(f.Size, 10, 64)
		modified, _ := time.Parse(time.RFC3339, f.ModifiedTime)
		files = append(files, GFile{
			ID:       f.ID,
			Name:     f.Name,
			MimeType: f.MimeType,
			Link:     f.WebViewLink,
			Size:     size,
			Modified: modified,
		})
	}

	return files, nil
}

func (c *GoogleDriveClient) GetFile(ctx context.Context, fileID string) (*GFile, error) {
	reqURL := fmt.Sprintf("https://www.googleapis.com/drive/v3/files/%s?fields=id,name,mimeType,webViewLink,size,modifiedTime", url.PathEscape(fileID))
	body, err := c.doRequest(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("gdrive: get file: %w", err)
	}

	var df driveFile
	if err := json.Unmarshal(body, &df); err != nil {
		return nil, fmt.Errorf("gdrive: unmarshal file: %w", err)
	}

	size, _ := strconv.ParseInt(df.Size, 10, 64)
	modified, _ := time.Parse(time.RFC3339, df.ModifiedTime)
	return &GFile{
		ID:       df.ID,
		Name:     df.Name,
		MimeType: df.MimeType,
		Link:     df.WebViewLink,
		Size:     size,
		Modified: modified,
	}, nil
}

func (c *GoogleDriveClient) GetFileContent(ctx context.Context, fileID string) ([]byte, error) {
	reqURL := fmt.Sprintf("https://www.googleapis.com/drive/v3/files/%s?alt=media", url.PathEscape(fileID))
	body, err := c.doRequest(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("gdrive: get file content: %w", err)
	}
	return body, nil
}

func (c *GoogleDriveClient) doRequest(ctx context.Context, method, reqURL string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized && c.refreshToken != "" {
		if refreshErr := c.refreshAccessToken(ctx); refreshErr != nil {
			return nil, fmt.Errorf("gdrive: auth refresh failed: %w", refreshErr)
		}
		return c.retryRequest(ctx, method, reqURL, body)
	}

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gdrive: HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return io.ReadAll(resp.Body)
}

func (c *GoogleDriveClient) retryRequest(ctx context.Context, method, reqURL string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gdrive: HTTP %d after refresh: %s", resp.StatusCode, string(respBody))
	}

	return io.ReadAll(resp.Body)
}

func (c *GoogleDriveClient) refreshAccessToken(ctx context.Context) error {
	params := url.Values{}
	params.Set("client_id", c.clientID)
	params.Set("client_secret", c.clientSecret)
	params.Set("refresh_token", c.refreshToken)
	params.Set("grant_type", "refresh_token")

	req, err := http.NewRequestWithContext(ctx, "POST", "https://oauth2.googleapis.com/token", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.URL.RawQuery = params.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("token refresh HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var tokenResp tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("decode token response: %w", err)
	}

	c.accessToken = tokenResp.AccessToken
	return nil
}
