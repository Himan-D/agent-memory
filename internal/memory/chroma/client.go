package chroma

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

// Client implements VectorStore using the Chroma REST API (v1).
type Client struct {
	cfg          config.ChromaConfig
	httpClient   *http.Client
	collectionID string
}

func NewClient(cfg config.ChromaConfig) (*Client, error) {
	if cfg.URL == "" {
		cfg.URL = "http://localhost:8000"
	}
	c := &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := c.ensureCollection(ctx); err != nil {
		return nil, fmt.Errorf("chroma: ensure collection: %w", err)
	}
	return c, nil
}

func (c *Client) Close() error { return nil }

func (c *Client) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.cfg.URL+"/api/v1/heartbeat", nil)
	if err != nil {
		return fmt.Errorf("chroma ping: build request: %w", err)
	}
	c.addAuth(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("chroma ping: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("chroma ping: status %d: %s", resp.StatusCode, body)
	}
	return nil
}

func (c *Client) ensureCollection(ctx context.Context) error {
	// Try to get the collection first.
	reqBody := map[string]interface{}{
		"name": c.cfg.Collection,
	}
	var getResp struct {
		ID string `json:"id"`
	}
	err := c.doRequest(ctx, http.MethodPost,
		"/api/v1/collections", reqBody, &getResp)
	if err != nil {
		// Collection may already exist — attempt to fetch it.
		var listResp []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}
		if err2 := c.doRequest(ctx, http.MethodGet,
			"/api/v1/collections", nil, &listResp); err2 != nil {
			return fmt.Errorf("ensure collection: %w", err)
		}
		for _, col := range listResp {
			if col.Name == c.cfg.Collection {
				c.collectionID = col.ID
				return nil
			}
		}
		return fmt.Errorf("ensure collection: %w", err)
	}
	c.collectionID = getResp.ID
	return nil
}

// addRequest mirrors the Chroma /collections/{id}/add JSON body.
type addRequest struct {
	IDs        []string                 `json:"ids"`
	Embeddings [][]float32              `json:"embeddings"`
	Documents  []string                 `json:"documents"`
	Metadatas  []map[string]interface{} `json:"metadatas"`
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
		// Chroma only accepts string/int/float metadata values — stringify everything.
		meta[k] = fmt.Sprintf("%v", v)
	}
	meta["entity_id"] = id
	meta["created_at"] = time.Now().Format(time.RFC3339)

	body := addRequest{
		IDs:        []string{pointID},
		Embeddings: [][]float32{embedding},
		Documents:  []string{text},
		Metadatas:  []map[string]interface{}{meta},
	}

	path := fmt.Sprintf("/api/v1/collections/%s/add", c.collectionID)
	if err := c.doRequest(ctx, http.MethodPost, path, body, nil); err != nil {
		return "", fmt.Errorf("chroma store embedding: %w", err)
	}
	return pointID, nil
}

// queryRequest mirrors the Chroma /collections/{id}/query JSON body.
type queryRequest struct {
	QueryEmbeddings [][]float32            `json:"query_embeddings"`
	NResults        int                    `json:"n_results"`
	Where           map[string]interface{} `json:"where,omitempty"`
	Include         []string               `json:"include"`
}

type queryResponse struct {
	IDs       [][]string                 `json:"ids"`
	Distances [][]float32                `json:"distances"`
	Documents [][]string                 `json:"documents"`
	Metadatas [][]map[string]interface{} `json:"metadatas"`
}

func (c *Client) Search(
	ctx context.Context,
	query []float32,
	limit int,
	threshold float32,
	filters map[string]interface{},
) ([]types.MemoryResult, error) {
	reqBody := queryRequest{
		QueryEmbeddings: [][]float32{query},
		NResults:        limit,
		Include:         []string{"distances", "documents", "metadatas"},
	}
	if len(filters) > 0 {
		// Chroma where filter: {"key": {"$eq": "value"}}
		where := make(map[string]interface{}, len(filters))
		for k, v := range filters {
			where[k] = map[string]interface{}{"$eq": fmt.Sprintf("%v", v)}
		}
		reqBody.Where = where
	}

	path := fmt.Sprintf("/api/v1/collections/%s/query", c.collectionID)
	var resp queryResponse
	if err := c.doRequest(ctx, http.MethodPost, path, reqBody, &resp); err != nil {
		return nil, fmt.Errorf("chroma search: %w", err)
	}

	if len(resp.IDs) == 0 || len(resp.IDs[0]) == 0 {
		return nil, nil
	}

	ids := resp.IDs[0]
	distances := resp.Distances[0]
	docs := resp.Documents[0]
	metas := resp.Metadatas[0]

	var results []types.MemoryResult
	for i, pid := range ids {
		// Chroma returns cosine distance (0=identical, 2=opposite); convert to similarity.
		var score float32
		if len(distances) > i {
			score = 1 - distances[i]/2
		}
		if score < threshold {
			continue
		}

		text := ""
		if len(docs) > i {
			text = docs[i]
		}
		meta := map[string]interface{}{}
		if len(metas) > i {
			meta = metas[i]
		}
		entityID := ""
		if eid, ok := meta["entity_id"].(string); ok {
			entityID = eid
		}

		results = append(results, types.MemoryResult{
			Entity: types.Entity{
				ID:         entityID,
				Properties: meta,
			},
			Score:    score,
			Text:     text,
			Source:   "chroma",
			MemoryID: pid,
		})
	}
	return results, nil
}

// updateMetadataRequest mirrors the Chroma /collections/{id}/update JSON body.
type updateMetadataRequest struct {
	IDs       []string                 `json:"ids"`
	Documents []string                 `json:"documents"`
	Metadatas []map[string]interface{} `json:"metadatas"`
}

func (c *Client) UpdateMemory(
	ctx context.Context,
	id string,
	text string,
	metadata map[string]interface{},
) error {
	meta := make(map[string]interface{}, len(metadata)+1)
	for k, v := range metadata {
		meta[k] = fmt.Sprintf("%v", v)
	}
	meta["updated_at"] = time.Now().Format(time.RFC3339)

	body := updateMetadataRequest{
		IDs:       []string{id},
		Documents: []string{text},
		Metadatas: []map[string]interface{}{meta},
	}
	path := fmt.Sprintf("/api/v1/collections/%s/update", c.collectionID)
	if err := c.doRequest(ctx, http.MethodPost, path, body, nil); err != nil {
		return fmt.Errorf("chroma update memory: %w", err)
	}
	return nil
}

// deleteRequest mirrors the Chroma /collections/{id}/delete JSON body.
type deleteRequest struct {
	IDs []string `json:"ids"`
}

func (c *Client) DeleteMemory(ctx context.Context, id string) error {
	body := deleteRequest{IDs: []string{id}}
	path := fmt.Sprintf("/api/v1/collections/%s/delete", c.collectionID)
	if err := c.doRequest(ctx, http.MethodPost, path, body, nil); err != nil {
		return fmt.Errorf("chroma delete memory: %w", err)
	}
	return nil
}

// updateVectorRequest mirrors the Chroma /collections/{id}/update JSON body with embeddings.
type updateVectorRequest struct {
	IDs        []string    `json:"ids"`
	Embeddings [][]float32 `json:"embeddings"`
}

func (c *Client) UpdateVector(ctx context.Context, id string, embedding []float32) error {
	body := updateVectorRequest{
		IDs:        []string{id},
		Embeddings: [][]float32{embedding},
	}
	path := fmt.Sprintf("/api/v1/collections/%s/update", c.collectionID)
	if err := c.doRequest(ctx, http.MethodPost, path, body, nil); err != nil {
		return fmt.Errorf("chroma update vector: %w", err)
	}
	return nil
}

func (c *Client) addAuth(req *http.Request) {
	if c.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	}
}

func (c *Client) doRequest(
	ctx context.Context,
	method string,
	path string,
	reqBody interface{},
	out interface{},
) error {
	var bodyReader io.Reader
	if reqBody != nil {
		data, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.cfg.URL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	c.addAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http %s %s: %w", method, path, err)
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
