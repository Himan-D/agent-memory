package pinecone

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"

	"agent-memory/internal/config"
	"agent-memory/internal/memory/types"
)

// Client implements VectorStore using the Pinecone REST API.
type Client struct {
	cfg        config.PineconeConfig
	httpClient *http.Client
}

func NewClient(cfg config.PineconeConfig) (*Client, error) {
	if cfg.IndexHost == "" {
		return nil, fmt.Errorf("pinecone: PINECONE_INDEX_HOST is required")
	}
	c := &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
	if err := c.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("pinecone: ping failed: %w", err)
	}
	return c, nil
}

func (c *Client) Close() error { return nil }

func (c *Client) Ping(ctx context.Context) error {
	// Use the describe_index_stats endpoint as a health check.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("https://%s/describe_index_stats", c.cfg.IndexHost), nil)
	if err != nil {
		return fmt.Errorf("pinecone ping: build request: %w", err)
	}
	req.Header.Set("Api-Key", c.cfg.APIKey)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("pinecone ping: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("pinecone ping: status %d: %s", resp.StatusCode, body)
	}
	return nil
}

// upsertRequest mirrors the Pinecone /vectors/upsert JSON body.
type upsertRequest struct {
	Vectors   []pineconeVector `json:"vectors"`
	Namespace string           `json:"namespace,omitempty"`
}

type pineconeVector struct {
	ID       string                 `json:"id"`
	Values   []float32              `json:"values"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

func (c *Client) StoreEmbedding(
	ctx context.Context,
	text string,
	id string,
	embedding []float32,
	metadata map[string]interface{},
) (string, error) {
	pointID := uuid.New().String()

	meta := make(map[string]interface{}, len(metadata)+2)
	for k, v := range metadata {
		meta[k] = v
	}
	meta["text"] = text
	meta["entity_id"] = id
	meta["created_at"] = time.Now().Format(time.RFC3339)

	body := upsertRequest{
		Vectors: []pineconeVector{
			{ID: pointID, Values: embedding, Metadata: meta},
		},
		Namespace: c.cfg.Namespace,
	}

	if err := c.doPost(ctx, "/vectors/upsert", body, nil); err != nil {
		return "", fmt.Errorf("pinecone store embedding: %w", err)
	}
	return pointID, nil
}

// queryRequest mirrors the Pinecone /query JSON body.
type queryRequest struct {
	Vector          []float32              `json:"vector"`
	TopK            int                    `json:"topK"`
	Namespace       string                 `json:"namespace,omitempty"`
	Filter          map[string]interface{} `json:"filter,omitempty"`
	IncludeMetadata bool                   `json:"includeMetadata"`
}

type queryResponse struct {
	Matches []struct {
		ID       string                 `json:"id"`
		Score    float32                `json:"score"`
		Metadata map[string]interface{} `json:"metadata"`
	} `json:"matches"`
}

func (c *Client) Search(
	ctx context.Context,
	query []float32,
	limit int,
	threshold float32,
	filters map[string]interface{},
) ([]types.MemoryResult, error) {
	reqBody := queryRequest{
		Vector:          query,
		TopK:            limit,
		Namespace:       c.cfg.Namespace,
		IncludeMetadata: true,
	}
	if len(filters) > 0 {
		reqBody.Filter = filters
	}

	var resp queryResponse
	if err := c.doPost(ctx, "/query", reqBody, &resp); err != nil {
		return nil, fmt.Errorf("pinecone search: %w", err)
	}

	var results []types.MemoryResult
	for _, m := range resp.Matches {
		if m.Score < threshold {
			continue
		}
		text := ""
		if t, ok := m.Metadata["text"].(string); ok {
			text = t
		}
		entityID := ""
		if eid, ok := m.Metadata["entity_id"].(string); ok {
			entityID = eid
		}
		results = append(results, types.MemoryResult{
			Entity: types.Entity{
				ID:         entityID,
				Properties: m.Metadata,
			},
			Score:    m.Score,
			Text:     text,
			Source:   "pinecone",
			MemoryID: m.ID,
		})
	}
	return results, nil
}

// updateRequest mirrors the Pinecone /vectors/update JSON body.
type updateRequest struct {
	ID          string                 `json:"id"`
	SetMetadata map[string]interface{} `json:"setMetadata,omitempty"`
	Namespace   string                 `json:"namespace,omitempty"`
}

func (c *Client) UpdateMemory(
	ctx context.Context,
	id string,
	text string,
	metadata map[string]interface{},
) error {
	meta := make(map[string]interface{}, len(metadata)+2)
	for k, v := range metadata {
		meta[k] = v
	}
	meta["text"] = text
	meta["last_accessed"] = time.Now().Format(time.RFC3339)

	body := updateRequest{
		ID:          id,
		SetMetadata: meta,
		Namespace:   c.cfg.Namespace,
	}
	if err := c.doPost(ctx, "/vectors/update", body, nil); err != nil {
		return fmt.Errorf("pinecone update memory: %w", err)
	}
	return nil
}

// deleteRequest mirrors the Pinecone /vectors/delete JSON body.
type deleteRequest struct {
	IDs       []string `json:"ids"`
	Namespace string   `json:"namespace,omitempty"`
}

func (c *Client) DeleteMemory(ctx context.Context, id string) error {
	body := deleteRequest{
		IDs:       []string{id},
		Namespace: c.cfg.Namespace,
	}
	if err := c.doPost(ctx, "/vectors/delete", body, nil); err != nil {
		return fmt.Errorf("pinecone delete memory: %w", err)
	}
	return nil
}

// vectorUpdateRequest mirrors the Pinecone /vectors/update JSON body with values.
type vectorUpdateRequest struct {
	ID        string    `json:"id"`
	Values    []float32 `json:"values"`
	Namespace string    `json:"namespace,omitempty"`
}

func (c *Client) UpdateVector(ctx context.Context, id string, embedding []float32) error {
	body := vectorUpdateRequest{
		ID:        id,
		Values:    embedding,
		Namespace: c.cfg.Namespace,
	}
	if err := c.doPost(ctx, "/vectors/update", body, nil); err != nil {
		return fmt.Errorf("pinecone update vector: %w", err)
	}
	return nil
}

// doPost sends a POST request to the Pinecone index host and optionally decodes the response.
func (c *Client) doPost(ctx context.Context, path string, reqBody interface{}, out interface{}) error {
	data, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("https://%s%s", c.cfg.IndexHost, path)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Api-Key", c.cfg.APIKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http post %s: %w", path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("status %d: %s", resp.StatusCode, body)
	}
	if out != nil && len(body) > 0 {
		if err := json.Unmarshal(body, out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

func (c *Client) BatchStoreEmbeddings(ctx context.Context, items []types.BatchEmbeddingItem) error {
	for _, item := range items {
		if _, err := c.StoreEmbedding(ctx, item.Text, item.ID, item.Embedding, item.Metadata); err != nil {
			return fmt.Errorf("pinecone batch store %s: %w", item.ID, err)
		}
	}
	return nil
}
