package vector

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"agent-memory/internal/config"
	"agent-memory/internal/memory/types"
)

type OpenSearchClient struct {
	baseURL string
	index   string
	client  *http.Client
	cfg     config.OpenSearchConfig
}

func NewOpenSearchClient(cfg config.OpenSearchConfig) (*OpenSearchClient, error) {
	c := &OpenSearchClient{
		baseURL: cfg.URL,
		index:   cfg.Index,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		cfg: cfg,
	}
	if err := c.ensureIndex(context.Background()); err != nil {
		return nil, fmt.Errorf("opensearch ensure index: %w", err)
	}
	return c, nil
}

func (c *OpenSearchClient) doRequest(method, path string, body interface{}) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal body: %w", err)
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.baseURL+path, reader)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.cfg.APIKey != "" {
		req.Header.Set("Authorization", "ApiKey "+c.cfg.APIKey)
	} else if c.cfg.Username != "" {
		req.SetBasicAuth(c.cfg.Username, c.cfg.Password)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("opensearch %s %s: status %d: %s", method, path, resp.StatusCode, string(data))
	}
	return data, nil
}

func (c *OpenSearchClient) ensureIndex(ctx context.Context) error {
	_, err := c.doRequest("GET", "/"+c.index, nil)
	if err == nil {
		return nil
	}
	mapping := map[string]interface{}{
		"settings": map[string]interface{}{
			"index.knn": true,
		},
		"mappings": map[string]interface{}{
			"properties": map[string]interface{}{
				"text": map[string]interface{}{
					"type": "text",
				},
				"entity_id": map[string]interface{}{
					"type": "keyword",
				},
				"tenant_id": map[string]interface{}{
					"type": "keyword",
				},
				"embedding": map[string]interface{}{
					"type":           "knn_vector",
					"dimension":      c.cfg.VectorSize,
					"method": map[string]interface{}{
						"name":       "hnsw",
						"space_type": "cosinesimil",
						"engine":     "nmslib",
						"parameters": map[string]interface{}{
							"ef_construction": 128,
							"m":              24,
						},
					},
				},
				"created_at": map[string]interface{}{
					"type": "date",
				},
				"last_accessed": map[string]interface{}{
					"type": "date",
				},
			},
		},
	}
	_, err = c.doRequest("PUT", "/"+c.index, mapping)
	if err != nil {
		return fmt.Errorf("create index: %w", err)
	}
	return nil
}

func (c *OpenSearchClient) StoreEmbedding(ctx context.Context, text string, id string, embedding []float32, metadata map[string]interface{}) (string, error) {
	doc := map[string]interface{}{
		"text":          text,
		"entity_id":     id,
		"embedding":     embedding,
		"created_at":    time.Now().Format(time.RFC3339),
		"last_accessed": time.Now().Format(time.RFC3339),
	}
	for k, v := range metadata {
		doc[k] = v
	}
	docID := fmt.Sprintf("%d", time.Now().UnixNano())
	_, err := c.doRequest("PUT", "/"+c.index+"/_doc/"+docID, doc)
	if err != nil {
		return "", fmt.Errorf("store embedding: %w", err)
	}
	return docID, nil
}

func (c *OpenSearchClient) search(ctx context.Context, query []float32, limit int, threshold float32, filters map[string]interface{}) ([]types.MemoryResult, error) {
	body := map[string]interface{}{
		"size": limit,
		"query": map[string]interface{}{
			"knn": map[string]interface{}{
				"embedding": map[string]interface{}{
					"vector": query,
					"k":      limit,
				},
			},
		},
		"min_score": threshold,
	}
	if len(filters) > 0 {
		boolQuery := map[string]interface{}{
			"must": []interface{}{
				map[string]interface{}{
					"knn": map[string]interface{}{
						"embedding": map[string]interface{}{
							"vector": query,
							"k":      limit,
						},
					},
				},
			},
			"filter": c.buildFilters(filters),
		}
		body["query"] = map[string]interface{}{"bool": boolQuery}
	}
	data, err := c.doRequest("POST", "/"+c.index+"/_search", body)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	var result struct {
		Hits struct {
			Hits []struct {
				ID     string                 `json:"_id"`
				Score  float64                `json:"_score"`
				Source map[string]interface{} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("unmarshal search result: %w", err)
	}
	var results []types.MemoryResult
	for _, hit := range result.Hits.Hits {
		text, _ := hit.Source["text"].(string)
		entityID, _ := hit.Source["entity_id"].(string)
		props := make(map[string]interface{})
		for k, v := range hit.Source {
			props[k] = v
		}
		results = append(results, types.MemoryResult{
			Entity: types.Entity{
				ID:         entityID,
				Properties: props,
			},
			Score:  float32(hit.Score),
			Text:   text,
			Source: "opensearch",
		})
	}
	return results, nil
}

func (c *OpenSearchClient) buildFilters(filters map[string]interface{}) []interface{} {
	var terms []interface{}
	for k, v := range filters {
		terms = append(terms, map[string]interface{}{
			"term": map[string]interface{}{k: v},
		})
	}
	return terms
}

func (c *OpenSearchClient) Search(ctx context.Context, query []float32, limit int, threshold float32, filters map[string]interface{}) ([]types.MemoryResult, error) {
	return c.search(ctx, query, limit, threshold, filters)
}

func (c *OpenSearchClient) SearchWithTenant(ctx context.Context, query []float32, limit int, threshold float32, filters map[string]interface{}, tenantID string) ([]types.MemoryResult, error) {
	if tenantID != "" {
		if filters == nil {
			filters = make(map[string]interface{})
		}
		filters["tenant_id"] = tenantID
	}
	return c.search(ctx, query, limit, threshold, filters)
}

func (c *OpenSearchClient) UpdateMemory(ctx context.Context, id string, text string, metadata map[string]interface{}) error {
	doc := map[string]interface{}{
		"doc": map[string]interface{}{
			"text":          text,
			"last_accessed": time.Now().Format(time.RFC3339),
		},
	}
	for k, v := range metadata {
		doc["doc"].(map[string]interface{})[k] = v
	}
	_, err := c.doRequest("POST", "/"+c.index+"/_update/"+id, doc)
	if err != nil {
		return fmt.Errorf("update memory: %w", err)
	}
	return nil
}

func (c *OpenSearchClient) DeleteMemory(ctx context.Context, id string) error {
	_, err := c.doRequest("DELETE", "/"+c.index+"/_doc/"+id, nil)
	if err != nil {
		return fmt.Errorf("delete memory: %w", err)
	}
	return nil
}

func (c *OpenSearchClient) UpdateVector(ctx context.Context, id string, embedding []float32) error {
	doc := map[string]interface{}{
		"doc": map[string]interface{}{
			"embedding": embedding,
		},
	}
	_, err := c.doRequest("POST", "/"+c.index+"/_update/"+id, doc)
	if err != nil {
		return fmt.Errorf("update vector: %w", err)
	}
	return nil
}

func (c *OpenSearchClient) Ping(ctx context.Context) error {
	data, err := c.doRequest("GET", "/_cluster/health", nil)
	if err != nil {
		return err
	}
	var health struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(data, &health); err != nil {
		return fmt.Errorf("parse health: %w", err)
	}
	if health.Status == "red" {
		return fmt.Errorf("opensearch cluster status: red")
	}
	return nil
}

func (c *OpenSearchClient) Close() error {
	c.client.CloseIdleConnections()
	return nil
}

func init() {
	_ = strconv.Itoa(0)
}
