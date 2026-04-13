package reranker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"agent-memory/internal/memory/types"
)

type CohereReranker struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

func NewCohereReranker(apiKey, baseURL, model string) (*CohereReranker, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("cohere API key required")
	}
	if baseURL == "" {
		baseURL = "https://api.cohere.ai"
	}
	if model == "" {
		model = "cohere/rerank-english-v2.0"
	}

	return &CohereReranker{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

func (c *CohereReranker) Name() string {
	return "cohere"
}

func (c *CohereReranker) Rerank(ctx context.Context, query string, results []types.MemoryResult, limit int) ([]types.MemoryResult, error) {
	if len(results) == 0 {
		return results, nil
	}

	docs := make([]string, len(results))
	for i, r := range results {
		docs[i] = r.Text
	}

	reqBody := map[string]interface{}{
		"query":            query,
		"documents":        docs,
		"top_n":            limit,
		"model":            c.model,
		"return_documents": false,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/rerank", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cohere-Version", "2024-11-01")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("cohere error %d: %s", resp.StatusCode, string(body))
	}

	var rerankResp struct {
		Results []struct {
			Index    int `json:"index"`
			Document struct {
				Text string `json:"text"`
			} `json:"document"`
			RelevanceScore float64 `json:"relevance_score"`
		} `json:"results"`
	}

	if err := json.Unmarshal(body, &rerankResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	reranked := make([]types.MemoryResult, 0, len(rerankResp.Results))
	seen := make(map[int]bool)

	for _, result := range rerankResp.Results {
		if result.Index < len(results) {
			reranked = append(reranked, results[result.Index])
			seen[result.Index] = true
		}
	}

	for i, r := range results {
		if !seen[i] {
			reranked = append(reranked, r)
		}
	}

	return reranked, nil
}

func (c *CohereReranker) Close() error {
	return nil
}
