package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client provides a small helper to call MCP endpoints.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	Timeout    time.Duration
}

func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL:    baseURL,
		HTTPClient: &http.Client{Timeout: 15 * time.Second},
		Timeout:    15 * time.Second,
	}
}

// ExecuteLinear posts a linear sequence of calls to /mcp/linear and returns
// the parsed LinearResponse.
func (c *Client) ExecuteLinear(ctx context.Context, calls []LinearCall) (*LinearResponse, error) {
	reqBody := LinearRequest{Calls: calls}
	b, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	url := c.BaseURL + "/mcp/linear"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("mcp error status %d: %s", resp.StatusCode, string(body))
	}

	var lr LinearResponse
	if err := json.Unmarshal(body, &lr); err != nil {
		return nil, err
	}
	return &lr, nil
}
