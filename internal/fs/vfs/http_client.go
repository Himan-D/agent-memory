package vfs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"agent-memory/internal/memory/types"
)

// HTTPClient implements ServiceInterface against the Hystersis REST API.
// Works on any OS without embedding Neo4j/Qdrant in the agentfs process.
type HTTPClient struct {
	baseURL  string
	apiKey   string
	tenantID string
	client   *http.Client
}

// NewHTTPClient creates an API-backed memory client for AgentFS.
func NewHTTPClient(baseURL, apiKey, tenantID string) *HTTPClient {
	baseURL = strings.TrimRight(baseURL, "/")
	if tenantID == "" {
		tenantID = "default"
	}
	return &HTTPClient{
		baseURL:  baseURL,
		apiKey:   apiKey,
		tenantID: tenantID,
		client:   &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *HTTPClient) do(ctx context.Context, method, path string, body any, out any) error {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, rdr)
	if err != nil {
		return err
	}
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}
	req.Header.Set("X-Tenant-ID", c.tenantID)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("api %s %s: %s: %s", method, path, resp.Status, strings.TrimSpace(string(data)))
	}
	if out != nil && len(data) > 0 {
		return json.Unmarshal(data, out)
	}
	return nil
}

func (c *HTTPClient) SearchMemories(ctx context.Context, req *types.SearchRequest) ([]types.MemoryResult, error) {
	if req == nil {
		return nil, fmt.Errorf("search request required")
	}
	req.TenantID = c.tenantID
	var results []types.MemoryResult
	if err := c.do(ctx, http.MethodPost, "/search", req, &results); err != nil {
		// Fallback: some servers wrap in {results:[]}
		var wrap struct {
			Results []types.MemoryResult `json:"results"`
		}
		if err2 := c.do(ctx, http.MethodPost, "/search", req, &wrap); err2 == nil {
			return wrap.Results, nil
		}
		return nil, err
	}
	return results, nil
}

func (c *HTTPClient) GetMemory(ctx context.Context, id string) (*types.Memory, error) {
	var mem types.Memory
	if err := c.do(ctx, http.MethodGet, "/memories/"+url.PathEscape(id), nil, &mem); err != nil {
		return nil, err
	}
	return &mem, nil
}

func (c *HTTPClient) CreateMemory(ctx context.Context, mem *types.Memory) (*types.Memory, error) {
	if mem.TenantID == "" {
		mem.TenantID = c.tenantID
	}
	var out types.Memory
	if err := c.do(ctx, http.MethodPost, "/memories", mem, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *HTTPClient) UpdateMemory(ctx context.Context, id, content string, meta map[string]interface{}) error {
	body := map[string]interface{}{"content": content}
	if meta != nil {
		body["metadata"] = meta
	}
	return c.do(ctx, http.MethodPut, "/memories/"+url.PathEscape(id), body, nil)
}

func (c *HTTPClient) DeleteMemory(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/memories/"+url.PathEscape(id), nil, nil)
}

func (c *HTTPClient) ListSkills(ctx context.Context, tenantID, domain string, lim, off int) ([]*types.Skill, error) {
	if lim <= 0 {
		lim = 100
	}
	q := fmt.Sprintf("/skills?limit=%d&offset=%d", lim, off)
	if domain != "" {
		q += "&domain=" + url.QueryEscape(domain)
	}
	var wrap struct {
		Skills []*types.Skill `json:"skills"`
	}
	if err := c.do(ctx, http.MethodGet, q, nil, &wrap); err != nil {
		return nil, err
	}
	return wrap.Skills, nil
}

func (c *HTTPClient) ListSessions(ctx context.Context, userID string) ([]*types.Session, error) {
	var wrap struct {
		Sessions []*types.Session `json:"sessions"`
	}
	if err := c.do(ctx, http.MethodGet, "/sessions", nil, &wrap); err != nil {
		return nil, err
	}
	return wrap.Sessions, nil
}

func (c *HTTPClient) GetEntity(ctx context.Context, id string) (*types.Entity, error) {
	var e types.Entity
	if err := c.do(ctx, http.MethodGet, "/entities/"+url.PathEscape(id), nil, &e); err != nil {
		return nil, err
	}
	return &e, nil
}

func (c *HTTPClient) GetMemoriesByTenant(ctx context.Context, tenantID string, limit int) ([]*types.Memory, error) {
	if limit <= 0 {
		limit = 500
	}
	var wrap struct {
		Memories []*types.Memory `json:"memories"`
	}
	path := fmt.Sprintf("/memories?limit=%d", limit)
	if err := c.do(ctx, http.MethodGet, path, nil, &wrap); err != nil {
		return nil, err
	}
	return wrap.Memories, nil
}

var _ ServiceInterface = (*HTTPClient)(nil)
