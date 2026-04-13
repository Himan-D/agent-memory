package vector

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"agent-memory/internal/memory/types"
)

type pineconeProvider struct {
	apiKey      string
	baseURL     string
	index       string
	dimension   int
	environment string
	cloud       string
	client      *http.Client
}

func newPineconeProvider(cfg *Config) (*pineconeProvider, error) {
	if cfg.Pinecone.APIKey == "" {
		return nil, fmt.Errorf("pinecone API key is required")
	}
	if cfg.Pinecone.Index == "" {
		cfg.Pinecone.Index = "agent-memory"
	}
	if cfg.Pinecone.Dimension == 0 {
		cfg.Pinecone.Dimension = 1536
	}
	if cfg.Pinecone.Environment == "" {
		cfg.Pinecone.Environment = "us-east-1"
	}
	if cfg.Pinecone.Cloud == "" {
		cfg.Pinecone.Cloud = "aws"
	}

	baseURL := fmt.Sprintf("https://%s-%s.svc.%s.pinecone.io",
		cfg.Pinecone.Index, cfg.Pinecone.Environment, cfg.Pinecone.Environment)

	p := &pineconeProvider{
		apiKey:      cfg.Pinecone.APIKey,
		baseURL:     baseURL,
		index:       cfg.Pinecone.Index,
		dimension:   cfg.Pinecone.Dimension,
		environment: cfg.Pinecone.Environment,
		cloud:       cfg.Pinecone.Cloud,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	return p, nil
}

func (p *pineconeProvider) Name() ProviderType { return ProviderPinecone }

func (p *pineconeProvider) StoreEmbedding(ctx context.Context, text string, id string, embedding []float32, meta map[string]interface{}) (string, error) {
	metadata := p.buildMetadata(text, meta)

	record := map[string]interface{}{
		"id":       id,
		"metadata": metadata,
	}

	if embedding != nil {
		record["values"] = embedding
	}

	vectors := []map[string]interface{}{record}

	url := fmt.Sprintf("%s/vectors/upsert", p.baseURL)
	jsonBody, err := json.Marshal(map[string]interface{}{"vectors": vectors})
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("ApiKey", p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("upsert request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("upsert failed (%d): %s", resp.StatusCode, string(body))
	}

	return id, nil
}

func (p *pineconeProvider) buildMetadata(text string, meta map[string]interface{}) map[string]interface{} {
	result := map[string]interface{}{
		"text": text,
	}

	for k, v := range meta {
		result[k] = v
	}

	return result
}

func (p *pineconeProvider) Search(ctx context.Context, query []float32, limit int, threshold float32, filters map[string]interface{}) ([]types.MemoryResult, error) {
	queryReq := map[string]interface{}{
		"vector":          query,
		"topK":            limit,
		"includeMetadata": true,
		"includeValues":   false,
	}

	if len(filters) > 0 {
		queryReq["filter"] = p.buildFilter(filters)
	}

	if threshold > 0 {
		queryReq["minScore"] = threshold
	}

	url := fmt.Sprintf("%s/query", p.baseURL)
	jsonBody, err := json.Marshal(queryReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("ApiKey", p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("query request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("query failed (%d): %s", resp.StatusCode, string(body))
	}

	var queryResp struct {
		Matches []struct {
			ID       string                 `json:"id"`
			Score    float64                `json:"score"`
			Metadata map[string]interface{} `json:"metadata"`
		} `json:"matches"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&queryResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	var results []types.MemoryResult
	for _, match := range queryResp.Matches {
		text := ""
		if t, ok := match.Metadata["text"].(string); ok {
			text = t
		}

		delete(match.Metadata, "text")

		results = append(results, types.MemoryResult{
			Entity: types.Entity{
				ID:         match.ID,
				Properties: match.Metadata,
			},
			Score:  float32(match.Score),
			Text:   text,
			Source: "pinecone",
		})
	}

	return results, nil
}

func (p *pineconeProvider) buildFilter(filters map[string]interface{}) map[string]interface{} {
	var conditions []map[string]interface{}
	for k, v := range filters {
		conditions = append(conditions, map[string]interface{}{
			"field": k,
			"op":    "$eq",
			"value": v,
		})
	}

	if len(conditions) == 0 {
		return nil
	}

	if len(conditions) == 1 {
		return conditions[0]
	}

	return map[string]interface{}{
		"$and": conditions,
	}
}

func (p *pineconeProvider) UpdateMemory(ctx context.Context, id string, text string, meta map[string]interface{}) error {
	_, err := p.StoreEmbedding(ctx, text, id, nil, meta)
	return err
}

func (p *pineconeProvider) DeleteMemory(ctx context.Context, id string) error {
	url := fmt.Sprintf("%s/vectors/delete", p.baseURL)

	deleteReq := map[string]interface{}{
		"ids": []string{id},
	}

	jsonBody, err := json.Marshal(deleteReq)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("ApiKey", p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("delete request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete failed (%d): %s", resp.StatusCode, string(body))
	}

	return nil
}

func (p *pineconeProvider) UpdateVector(ctx context.Context, id string, embedding []float32) error {
	url := fmt.Sprintf("%s/vectors/upsert", p.baseURL)

	vectors := []map[string]interface{}{
		{
			"id":     id,
			"values": embedding,
		},
	}

	jsonBody, err := json.Marshal(map[string]interface{}{"vectors": vectors})
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("ApiKey", p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("upsert request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upsert failed (%d): %s", resp.StatusCode, string(body))
	}

	return nil
}

func (p *pineconeProvider) DeleteByFilter(ctx context.Context, filter map[string]interface{}) (int, error) {
	url := fmt.Sprintf("%s/vectors/delete", p.baseURL)

	deleteReq := map[string]interface{}{}
	if filter != nil {
		deleteReq["filter"] = p.buildFilter(filter)
	} else {
		deleteReq["deleteAll"] = true
	}

	jsonBody, err := json.Marshal(deleteReq)
	if err != nil {
		return 0, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return 0, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("ApiKey", p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("delete request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("delete failed (%d): %s", resp.StatusCode, string(body))
	}

	return 0, nil
}

func (p *pineconeProvider) Ping(ctx context.Context) error {
	url := fmt.Sprintf("%s/stats", p.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("ApiKey", p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("stats request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("stats failed (%d): %s", resp.StatusCode, string(body))
	}

	var stats struct {
		Dimension        int `json:"dimension"`
		TotalVectorCount int `json:"totalVectorCount"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	if stats.Dimension != p.dimension {
		return fmt.Errorf("dimension mismatch: expected %d, got %d", p.dimension, stats.Dimension)
	}

	return nil
}

func (p *pineconeProvider) Close() error {
	return nil
}

type weaviateProvider struct {
	url       string
	apiKey    string
	className string
	dimension int
	client    *http.Client
}

func newWeaviateProvider(cfg *Config) (*weaviateProvider, error) {
	if cfg.Weaviate.URL == "" {
		return nil, fmt.Errorf("weaviate URL is required")
	}
	if cfg.Weaviate.ClassName == "" {
		cfg.Weaviate.ClassName = "AgentMemory"
	}
	if cfg.Weaviate.Dimension == 0 {
		cfg.Weaviate.Dimension = 1536
	}

	return &weaviateProvider{
		url:       cfg.Weaviate.URL,
		apiKey:    cfg.Weaviate.APIKey,
		className: cfg.Weaviate.ClassName,
		dimension: cfg.Weaviate.Dimension,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

func (p *weaviateProvider) Name() ProviderType { return ProviderWeaviate }

func (p *weaviateProvider) StoreEmbedding(ctx context.Context, text string, id string, embedding []float32, meta map[string]interface{}) (string, error) {
	properties := map[string]interface{}{
		"text": text,
	}
	for k, v := range meta {
		properties[k] = v
	}

	doc := map[string]interface{}{
		"class":      p.className,
		"id":         id,
		"vector":     embedding,
		"properties": properties,
	}

	body, err := json.Marshal(doc)
	if err != nil {
		return "", fmt.Errorf("marshal document: %w", err)
	}

	url := fmt.Sprintf("%s/v1/objects", p.url)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("create object: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("create failed (%d): %s", resp.StatusCode, string(respBody))
	}

	return id, nil
}

func (p *weaviateProvider) Search(ctx context.Context, query []float32, limit int, threshold float32, filters map[string]interface{}) ([]types.MemoryResult, error) {
	nearVector := map[string]interface{}{
		"vector": query,
	}

	if threshold > 0 {
		nearVector["certainty"] = threshold
	}

	searchReq := map[string]interface{}{
		"query": map[string]interface{}{
			"nearVector": nearVector,
			"limit":      limit,
		},
		"fields": []string{"text", "metadata"},
	}

	if len(filters) > 0 {
		searchReq["query"].(map[string]interface{})["where"] = p.buildFilter(filters)
	}

	body, err := json.Marshal(searchReq)
	if err != nil {
		return nil, fmt.Errorf("marshal search query: %w", err)
	}

	url := fmt.Sprintf("%s/v1/objects", p.url)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Data struct {
			Objects []struct {
				ID         string                 `json:"id"`
				Properties map[string]interface{} `json:"properties"`
				Metadata   struct {
					Certainty float32 `json:"certainty"`
				} `json:"metadata"`
			} `json:"objects"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	var results []types.MemoryResult
	for _, obj := range result.Data.Objects {
		text, _ := obj.Properties["text"].(string)
		results = append(results, types.MemoryResult{
			MemoryID: obj.ID,
			Text:     text,
			Score:    obj.Metadata.Certainty,
			Source:   "weaviate",
		})
	}

	return results, nil
}

func (p *weaviateProvider) buildFilter(filters map[string]interface{}) map[string]interface{} {
	var conditions []map[string]interface{}
	for k, v := range filters {
		conditions = append(conditions, map[string]interface{}{
			"path":        []string{k},
			"operator":    "Equal",
			"valueString": v,
		})
	}

	if len(conditions) == 1 {
		return conditions[0]
	}

	return map[string]interface{}{
		"operator": "And",
		"operands": conditions,
	}
}

func (p *weaviateProvider) UpdateMemory(ctx context.Context, id string, text string, meta map[string]interface{}) error {
	properties := map[string]interface{}{
		"text": text,
	}
	for k, v := range meta {
		properties[k] = v
	}

	doc := map[string]interface{}{
		"properties": properties,
	}

	body, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshal update: %w", err)
	}

	url := fmt.Sprintf("%s/v1/objects/%s", p.url, id)

	req, err := http.NewRequestWithContext(ctx, "PATCH", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("update request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("update failed (%d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (p *weaviateProvider) DeleteMemory(ctx context.Context, id string) error {
	url := fmt.Sprintf("%s/v1/objects/%s", p.url, id)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("delete request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 && resp.StatusCode != 404 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete failed (%d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (p *weaviateProvider) UpdateVector(ctx context.Context, id string, embedding []float32) error {
	doc := map[string]interface{}{
		"vector": embedding,
	}

	body, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshal update: %w", err)
	}

	url := fmt.Sprintf("%s/v1/objects/%s", p.url, id)

	req, err := http.NewRequestWithContext(ctx, "PATCH", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("update vector request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("update vector failed (%d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (p *weaviateProvider) DeleteByFilter(ctx context.Context, filter map[string]interface{}) (int, error) {
	whereFilter := p.buildFilter(filter)

	searchReq := map[string]interface{}{
		"query": map[string]interface{}{
			"where": whereFilter,
			"limit": 100,
		},
	}

	body, err := json.Marshal(searchReq)
	if err != nil {
		return 0, fmt.Errorf("marshal search query: %w", err)
	}

	url := fmt.Sprintf("%s/v1/objects", p.url)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return 0, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("search request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("search failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Data struct {
			Objects []struct {
				ID string `json:"id"`
			} `json:"objects"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("decode response: %w", err)
	}

	deleted := 0
	for _, obj := range result.Data.Objects {
		if err := p.DeleteMemory(ctx, obj.ID); err == nil {
			deleted++
		}
	}

	return deleted, nil
}

func (p *weaviateProvider) Ping(ctx context.Context) error {
	url := fmt.Sprintf("%s/v1/meta", p.url)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("ping request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("ping failed (%d)", resp.StatusCode)
	}

	return nil
}

func (p *weaviateProvider) Close() error {
	return nil
}

type chromaProvider struct {
	url        string
	apiKey     string
	collection string
	dimension  int
	client     *http.Client
}

func newChromaProvider(cfg *Config) (*chromaProvider, error) {
	if cfg.Chroma.URL == "" {
		cfg.Chroma.URL = "http://localhost:8000"
	}
	if cfg.Chroma.Collection == "" {
		cfg.Chroma.Collection = "agent_memory"
	}
	if cfg.Chroma.Dimension == 0 {
		cfg.Chroma.Dimension = 1536
	}

	return &chromaProvider{
		url:        cfg.Chroma.URL,
		apiKey:     cfg.Chroma.APIKey,
		collection: cfg.Chroma.Collection,
		dimension:  cfg.Chroma.Dimension,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

func (p *chromaProvider) Name() ProviderType { return ProviderChroma }

func (p *chromaProvider) StoreEmbedding(ctx context.Context, text string, id string, embedding []float32, meta map[string]interface{}) (string, error) {
	metadata := map[string]interface{}{
		"text": text,
	}
	for k, v := range meta {
		metadata[k] = v
	}

	record := map[string]interface{}{
		"id":        id,
		"embedding": embedding,
		"metadata":  metadata,
		"documents": []string{text},
	}

	body, err := json.Marshal(map[string]interface{}{
		"ids":        []string{id},
		"embeddings": [][]float32{embedding},
		"metadatas":  []map[string]interface{}{metadata},
		"documents":  []string{text},
	})
	if err != nil {
		return "", fmt.Errorf("marshal document: %w", err)
	}

	url := fmt.Sprintf("%s/v1/collections/%s/records", p.url, p.collection)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("add record: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("add failed (%d): %s", resp.StatusCode, string(respBody))
	}

	_ = record

	return id, nil
}

func (p *chromaProvider) Search(ctx context.Context, query []float32, limit int, threshold float32, filters map[string]interface{}) ([]types.MemoryResult, error) {
	queryReq := map[string]interface{}{
		"query_embeddings": [][]float32{query},
		"n_results":        limit,
		"include":          []string{"metadatas", "documents", "distances"},
	}

	if len(filters) > 0 {
		queryReq["where"] = p.buildFilter(filters)
	}

	body, err := json.Marshal(queryReq)
	if err != nil {
		return nil, fmt.Errorf("marshal query: %w", err)
	}

	url := fmt.Sprintf("%s/v1/collections/%s/query", p.url, p.collection)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("query request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("query failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		IDs       [][]string                 `json:"ids"`
		Documents [][]string                 `json:"documents"`
		Metadatas [][]map[string]interface{} `json:"metadatas"`
		Distances [][]float64                `json:"distances"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	var results []types.MemoryResult
	for i, ids := range result.IDs {
		for j, id := range ids {
			score := 1.0
			if len(result.Distances) > i && len(result.Distances[i]) > j {
				score = 1.0 - result.Distances[i][j]
			}

			text := ""
			if len(result.Documents) > i && len(result.Documents[i]) > j {
				text = result.Documents[i][j]
			}

			results = append(results, types.MemoryResult{
				MemoryID: id,
				Text:     text,
				Score:    float32(score),
				Source:   "chroma",
			})
		}
	}

	return results, nil
}

func (p *chromaProvider) buildFilter(filters map[string]interface{}) map[string]interface{} {
	return filters
}

func (p *chromaProvider) UpdateMemory(ctx context.Context, id string, text string, meta map[string]interface{}) error {
	metadata := map[string]interface{}{
		"text": text,
	}
	for k, v := range meta {
		metadata[k] = v
	}

	updateReq := map[string]interface{}{
		"ids":       []string{id},
		"metadatas": []map[string]interface{}{metadata},
		"documents": []string{text},
	}

	body, err := json.Marshal(updateReq)
	if err != nil {
		return fmt.Errorf("marshal update: %w", err)
	}

	url := fmt.Sprintf("%s/v1/collections/%s/records", p.url, p.collection)

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("update request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("update failed (%d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (p *chromaProvider) DeleteMemory(ctx context.Context, id string) error {
	deleteReq := map[string]interface{}{
		"ids": []string{id},
	}

	body, err := json.Marshal(deleteReq)
	if err != nil {
		return fmt.Errorf("marshal delete: %w", err)
	}

	url := fmt.Sprintf("%s/v1/collections/%s/records", p.url, p.collection)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("delete request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete failed (%d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (p *chromaProvider) UpdateVector(ctx context.Context, id string, embedding []float32) error {
	updateReq := map[string]interface{}{
		"ids":        []string{id},
		"embeddings": [][]float32{embedding},
	}

	body, err := json.Marshal(updateReq)
	if err != nil {
		return fmt.Errorf("marshal update: %w", err)
	}

	url := fmt.Sprintf("%s/v1/collections/%s/records", p.url, p.collection)

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("update vector request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("update vector failed (%d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (p *chromaProvider) DeleteByFilter(ctx context.Context, filter map[string]interface{}) (int, error) {
	deleteReq := map[string]interface{}{}

	if len(filter) > 0 {
		deleteReq["where"] = filter
	}

	body, err := json.Marshal(deleteReq)
	if err != nil {
		return 0, fmt.Errorf("marshal delete: %w", err)
	}

	url := fmt.Sprintf("%s/v1/collections/%s/records", p.url, p.collection)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, bytes.NewBuffer(body))
	if err != nil {
		return 0, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("delete request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("delete failed (%d): %s", resp.StatusCode, string(respBody))
	}

	return 1, nil
}

func (p *chromaProvider) Ping(ctx context.Context) error {
	url := fmt.Sprintf("%s/v1/heartbeat", p.url)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("ping request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("ping failed (%d)", resp.StatusCode)
	}

	return nil
}

func (p *chromaProvider) Close() error {
	return nil
}

type pgvectorProvider struct {
	host      string
	port      int
	user      string
	password  string
	database  string
	dimension int
}

func newPgvectorProvider(cfg *Config) (*pgvectorProvider, error) {
	return &pgvectorProvider{
		host:      cfg.Pgvector.Host,
		port:      cfg.Pgvector.Port,
		user:      cfg.Pgvector.User,
		password:  cfg.Pgvector.Password,
		database:  cfg.Pgvector.Database,
		dimension: cfg.Pgvector.Dimension,
	}, nil
}

func (p *pgvectorProvider) Name() ProviderType { return ProviderPgvector }

func (p *pgvectorProvider) StoreEmbedding(ctx context.Context, text string, id string, embedding []float32, meta map[string]interface{}) (string, error) {
	return id, nil
}

func (p *pgvectorProvider) Search(ctx context.Context, query []float32, limit int, threshold float32, filters map[string]interface{}) ([]types.MemoryResult, error) {
	return []types.MemoryResult{}, nil
}

func (p *pgvectorProvider) UpdateMemory(ctx context.Context, id string, text string, meta map[string]interface{}) error {
	return nil
}

func (p *pgvectorProvider) DeleteMemory(ctx context.Context, id string) error {
	return nil
}

func (p *pgvectorProvider) UpdateVector(ctx context.Context, id string, embedding []float32) error {
	return nil
}

func (p *pgvectorProvider) DeleteByFilter(ctx context.Context, filter map[string]interface{}) (int, error) {
	return 0, nil
}

func (p *pgvectorProvider) Ping(ctx context.Context) error {
	return nil
}

func (p *pgvectorProvider) Close() error {
	return nil
}

type milvusProvider struct {
	url        string
	apiKey     string
	collection string
}

func newMilvusProvider(cfg *Config) (*milvusProvider, error) {
	return &milvusProvider{
		url:        cfg.Milvus.URL,
		apiKey:     cfg.Milvus.APIKey,
		collection: cfg.Milvus.Collection,
	}, nil
}

func (p *milvusProvider) Name() ProviderType { return ProviderMilvus }

func (p *milvusProvider) StoreEmbedding(ctx context.Context, text string, id string, embedding []float32, meta map[string]interface{}) (string, error) {
	return id, nil
}

func (p *milvusProvider) Search(ctx context.Context, query []float32, limit int, threshold float32, filters map[string]interface{}) ([]types.MemoryResult, error) {
	return []types.MemoryResult{}, nil
}

func (p *milvusProvider) UpdateMemory(ctx context.Context, id string, text string, meta map[string]interface{}) error {
	return nil
}

func (p *milvusProvider) DeleteMemory(ctx context.Context, id string) error {
	return nil
}

func (p *milvusProvider) UpdateVector(ctx context.Context, id string, embedding []float32) error {
	return nil
}

func (p *milvusProvider) DeleteByFilter(ctx context.Context, filter map[string]interface{}) (int, error) {
	return 0, nil
}

func (p *milvusProvider) Ping(ctx context.Context) error {
	return nil
}

func (p *milvusProvider) Close() error {
	return nil
}

type elasticProvider struct {
	addresses []string
	apiKey    string
	index     string
	client    *http.Client
}

func newElasticProvider(cfg *Config) (*elasticProvider, error) {
	addresses := cfg.Elastic.Addresses
	if len(addresses) == 0 {
		addresses = []string{"http://localhost:9200"}
	}
	if cfg.Elastic.Index == "" {
		cfg.Elastic.Index = "agent-memory"
	}

	return &elasticProvider{
		addresses: addresses,
		apiKey:    cfg.Elastic.APIKey,
		index:     cfg.Elastic.Index,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

func (p *elasticProvider) Name() ProviderType { return ProviderElastic }

func (p *elasticProvider) StoreEmbedding(ctx context.Context, text string, id string, embedding []float32, meta map[string]interface{}) (string, error) {
	doc := map[string]interface{}{
		"text":      text,
		"embedding": embedding,
		"metadata":  meta,
	}

	body, err := json.Marshal(doc)
	if err != nil {
		return "", fmt.Errorf("marshal document: %w", err)
	}

	url := fmt.Sprintf("%s/%s/_doc/%s", p.addresses[0], p.index, id)

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "ApiKey "+p.apiKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("index document: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("index failed (%d): %s", resp.StatusCode, string(respBody))
	}

	return id, nil
}

func (p *elasticProvider) Search(ctx context.Context, query []float32, limit int, threshold float32, filters map[string]interface{}) ([]types.MemoryResult, error) {
	knnQuery := map[string]interface{}{
		"field":        "embedding",
		"query_vector": query,
		"k":            limit,
		"boost":        1.0,
	}

	if len(filters) > 0 {
		knnQuery["filter"] = p.buildFilter(filters)
	}

	searchBody := map[string]interface{}{
		"knn":     knnQuery,
		"_source": []string{"text", "metadata"},
	}

	body, err := json.Marshal(searchBody)
	if err != nil {
		return nil, fmt.Errorf("marshal search query: %w", err)
	}

	url := fmt.Sprintf("%s/%s/_search", p.addresses[0], p.index)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "ApiKey "+p.apiKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unmarshal response: %w, body: %s", err, string(respBody))
	}

	var results []types.MemoryResult

	if hits, ok := result["hits"].(map[string]interface{}); ok {
		if hitsArray, ok := hits["hits"].([]interface{}); ok {
			for _, hit := range hitsArray {
				if h, ok := hit.(map[string]interface{}); ok {
					source, _ := h["_source"].(map[string]interface{})
					text, _ := source["text"].(string)
					score := 0.0
					if s, ok := h["_score"].(float64); ok {
						score = s
					}

					memResult := types.MemoryResult{
						Text:  text,
						Score: float32(score),
					}
					if id, ok := h["_id"].(string); ok {
						memResult.MemoryID = id
					}
					results = append(results, memResult)
				}
			}
		}
	}

	return results, nil
}

func (p *elasticProvider) buildFilter(filters map[string]interface{}) map[string]interface{} {
	var must []map[string]interface{}
	for k, v := range filters {
		must = append(must, map[string]interface{}{
			"match": map[string]interface{}{
				k: v,
			},
		})
	}
	return map[string]interface{}{
		"bool": map[string]interface{}{
			"must": must,
		},
	}
}

func (p *elasticProvider) UpdateMemory(ctx context.Context, id string, text string, meta map[string]interface{}) error {
	doc := map[string]interface{}{
		"doc": map[string]interface{}{
			"text":     text,
			"metadata": meta,
		},
	}

	body, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshal update: %w", err)
	}

	url := fmt.Sprintf("%s/%s/_update/%s", p.addresses[0], p.index, id)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "ApiKey "+p.apiKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("update request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("update failed (%d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (p *elasticProvider) DeleteMemory(ctx context.Context, id string) error {
	url := fmt.Sprintf("%s/%s/_doc/%s", p.addresses[0], p.index, id)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	if p.apiKey != "" {
		req.Header.Set("Authorization", "ApiKey "+p.apiKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("delete request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 && resp.StatusCode != 404 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete failed (%d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (p *elasticProvider) UpdateVector(ctx context.Context, id string, embedding []float32) error {
	doc := map[string]interface{}{
		"doc": map[string]interface{}{
			"embedding": embedding,
		},
	}

	body, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshal update: %w", err)
	}

	url := fmt.Sprintf("%s/%s/_update/%s", p.addresses[0], p.index, id)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "ApiKey "+p.apiKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("update vector request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("update vector failed (%d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (p *elasticProvider) DeleteByFilter(ctx context.Context, filter map[string]interface{}) (int, error) {
	deleteQuery := map[string]interface{}{
		"query": p.buildFilter(filter),
	}

	body, err := json.Marshal(deleteQuery)
	if err != nil {
		return 0, fmt.Errorf("marshal delete query: %w", err)
	}

	url := fmt.Sprintf("%s/%s/_delete_by_query", p.addresses[0], p.index)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return 0, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "ApiKey "+p.apiKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("delete by query request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("delete by query failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		respBody, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("unmarshal response: %w, body: %s", err, string(respBody))
	}

	deleted := 0
	if d, ok := result["deleted"].(float64); ok {
		deleted = int(d)
	}

	return deleted, nil
}

func (p *elasticProvider) Ping(ctx context.Context) error {
	url := fmt.Sprintf("%s/_cluster/health", p.addresses[0])

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	if p.apiKey != "" {
		req.Header.Set("Authorization", "ApiKey "+p.apiKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("ping request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("ping failed (%d)", resp.StatusCode)
	}

	return nil
}

func (p *elasticProvider) Close() error {
	return nil
}

type vespaProvider struct {
	url         string
	apiKey      string
	zone        string
	application string
}

func newVespaProvider(cfg *Config) (*vespaProvider, error) {
	return &vespaProvider{
		url:         cfg.Vespa.URL,
		apiKey:      cfg.Vespa.APIKey,
		zone:        cfg.Vespa.Zone,
		application: cfg.Vespa.Application,
	}, nil
}

func (p *vespaProvider) Name() ProviderType { return ProviderVespa }

func (p *vespaProvider) StoreEmbedding(ctx context.Context, text string, id string, embedding []float32, meta map[string]interface{}) (string, error) {
	return id, nil
}

func (p *vespaProvider) Search(ctx context.Context, query []float32, limit int, threshold float32, filters map[string]interface{}) ([]types.MemoryResult, error) {
	return []types.MemoryResult{}, nil
}

func (p *vespaProvider) UpdateMemory(ctx context.Context, id string, text string, meta map[string]interface{}) error {
	return nil
}

func (p *vespaProvider) DeleteMemory(ctx context.Context, id string) error {
	return nil
}

func (p *vespaProvider) UpdateVector(ctx context.Context, id string, embedding []float32) error {
	return nil
}

func (p *vespaProvider) DeleteByFilter(ctx context.Context, filter map[string]interface{}) (int, error) {
	return 0, nil
}

func (p *vespaProvider) Ping(ctx context.Context) error {
	return nil
}

func (p *vespaProvider) Close() error {
	return nil
}

type redisProvider struct {
	addr      string
	password  string
	db        int
	dimension int
}

func newRedisProvider(cfg *Config) (*redisProvider, error) {
	return &redisProvider{
		addr:      cfg.Redis.Addr,
		password:  cfg.Redis.Password,
		db:        cfg.Redis.DB,
		dimension: cfg.Redis.Dimension,
	}, nil
}

func (p *redisProvider) Name() ProviderType { return ProviderRedis }

func (p *redisProvider) StoreEmbedding(ctx context.Context, text string, id string, embedding []float32, meta map[string]interface{}) (string, error) {
	return id, nil
}

func (p *redisProvider) Search(ctx context.Context, query []float32, limit int, threshold float32, filters map[string]interface{}) ([]types.MemoryResult, error) {
	return []types.MemoryResult{}, nil
}

func (p *redisProvider) UpdateMemory(ctx context.Context, id string, text string, meta map[string]interface{}) error {
	return nil
}

func (p *redisProvider) DeleteMemory(ctx context.Context, id string) error {
	return nil
}

func (p *redisProvider) UpdateVector(ctx context.Context, id string, embedding []float32) error {
	return nil
}

func (p *redisProvider) DeleteByFilter(ctx context.Context, filter map[string]interface{}) (int, error) {
	return 0, nil
}

func (p *redisProvider) Ping(ctx context.Context) error {
	return nil
}

func (p *redisProvider) Close() error {
	return nil
}

type mongoProvider struct {
	uri        string
	apiKey     string
	database   string
	collection string
}

func newMongoProvider(cfg *Config) (*mongoProvider, error) {
	return &mongoProvider{
		uri:        cfg.Mongo.URI,
		apiKey:     cfg.Mongo.APIKey,
		database:   cfg.Mongo.Database,
		collection: cfg.Mongo.Collection,
	}, nil
}

func (p *mongoProvider) Name() ProviderType { return ProviderMongo }

func (p *mongoProvider) StoreEmbedding(ctx context.Context, text string, id string, embedding []float32, meta map[string]interface{}) (string, error) {
	return id, nil
}

func (p *mongoProvider) Search(ctx context.Context, query []float32, limit int, threshold float32, filters map[string]interface{}) ([]types.MemoryResult, error) {
	return []types.MemoryResult{}, nil
}

func (p *mongoProvider) UpdateMemory(ctx context.Context, id string, text string, meta map[string]interface{}) error {
	return nil
}

func (p *mongoProvider) DeleteMemory(ctx context.Context, id string) error {
	return nil
}

func (p *mongoProvider) UpdateVector(ctx context.Context, id string, embedding []float32) error {
	return nil
}

func (p *mongoProvider) DeleteByFilter(ctx context.Context, filter map[string]interface{}) (int, error) {
	return 0, nil
}

func (p *mongoProvider) Ping(ctx context.Context) error {
	return nil
}

func (p *mongoProvider) Close() error {
	return nil
}

type azureSearchProvider struct {
	url       string
	apiKey    string
	indexName string
	dimension int
	client    *http.Client
}

func newAzureSearchProvider(cfg *Config) (*azureSearchProvider, error) {
	if cfg.AzureSearch.URL == "" {
		return nil, fmt.Errorf("Azure Search URL is required")
	}
	if cfg.AzureSearch.IndexName == "" {
		cfg.AzureSearch.IndexName = "agent-memory"
	}
	if cfg.AzureSearch.Dimension == 0 {
		cfg.AzureSearch.Dimension = 1536
	}

	return &azureSearchProvider{
		url:       cfg.AzureSearch.URL,
		apiKey:    cfg.AzureSearch.APIKey,
		indexName: cfg.AzureSearch.IndexName,
		dimension: cfg.AzureSearch.Dimension,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

func (p *azureSearchProvider) Name() ProviderType { return ProviderAzureSearch }

func (p *azureSearchProvider) StoreEmbedding(ctx context.Context, text string, id string, embedding []float32, meta map[string]interface{}) (string, error) {
	doc := map[string]interface{}{
		"id":        id,
		"text":      text,
		"embedding": embedding,
		"metadata":  meta,
	}

	body, err := json.Marshal(doc)
	if err != nil {
		return "", fmt.Errorf("marshal document: %w", err)
	}

	url := fmt.Sprintf("%s/indexes/%s/docs/index?api-version=2023-07-01", p.url, p.indexName)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("index document: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("index failed (%d): %s", resp.StatusCode, string(respBody))
	}

	return id, nil
}

func (p *azureSearchProvider) Search(ctx context.Context, query []float32, limit int, threshold float32, filters map[string]interface{}) ([]types.MemoryResult, error) {
	searchReq := map[string]interface{}{
		"vectors": []map[string]interface{}{
			{
				"vector":    query,
				"k":         limit,
				"fieldName": "embedding",
			},
		},
		"select": "id,text,metadata",
	}

	if len(filters) > 0 {
		searchReq["filter"] = p.buildFilter(filters)
	}

	body, err := json.Marshal(searchReq)
	if err != nil {
		return nil, fmt.Errorf("marshal search query: %w", err)
	}

	url := fmt.Sprintf("%s/indexes/%s/docs/search?api-version=2023-07-01", p.url, p.indexName)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unmarshal response: %w, body: %s", err, string(respBody))
	}

	var results []types.MemoryResult

	if values, ok := result["value"].([]interface{}); ok {
		for _, v := range values {
			if val, ok := v.(map[string]interface{}); ok {
				text, _ := val["text"].(string)
				id, _ := val["id"].(string)
				score := 0.0
				if s, ok := val["@search.score"].(float64); ok {
					score = s
				}

				results = append(results, types.MemoryResult{
					Text:     text,
					Score:    float32(score),
					MemoryID: id,
				})
			}
		}
	}

	return results, nil
}

func (p *azureSearchProvider) buildFilter(filters map[string]interface{}) string {
	var conditions []string
	for k, v := range filters {
		conditions = append(conditions, fmt.Sprintf("%s eq '%v'", k, v))
	}
	return strings.Join(conditions, " and ")
}

func (p *azureSearchProvider) UpdateMemory(ctx context.Context, id string, text string, meta map[string]interface{}) error {
	doc := map[string]interface{}{
		"@search.action": "merge",
		"id":             id,
		"text":           text,
		"metadata":       meta,
	}

	body, err := json.Marshal([]map[string]interface{}{doc})
	if err != nil {
		return fmt.Errorf("marshal update: %w", err)
	}

	url := fmt.Sprintf("%s/indexes/%s/docs/index?api-version=2023-07-01", p.url, p.indexName)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("update request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("update failed (%d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (p *azureSearchProvider) DeleteMemory(ctx context.Context, id string) error {
	doc := map[string]interface{}{
		"@search.action": "delete",
		"id":             id,
	}

	body, err := json.Marshal([]map[string]interface{}{doc})
	if err != nil {
		return fmt.Errorf("marshal delete: %w", err)
	}

	url := fmt.Sprintf("%s/indexes/%s/docs/index?api-version=2023-07-01", p.url, p.indexName)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("delete request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 && resp.StatusCode != 404 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete failed (%d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (p *azureSearchProvider) UpdateVector(ctx context.Context, id string, embedding []float32) error {
	doc := map[string]interface{}{
		"@search.action": "merge",
		"id":             id,
		"embedding":      embedding,
	}

	body, err := json.Marshal([]map[string]interface{}{doc})
	if err != nil {
		return fmt.Errorf("marshal update: %w", err)
	}

	url := fmt.Sprintf("%s/indexes/%s/docs/index?api-version=2023-07-01", p.url, p.indexName)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("update vector request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("update vector failed (%d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (p *azureSearchProvider) DeleteByFilter(ctx context.Context, filter map[string]interface{}) (int, error) {
	deleteReq := map[string]interface{}{
		"@search.action": "delete",
	}

	for k, v := range filter {
		deleteReq[k] = v
	}

	body, err := json.Marshal([]map[string]interface{}{deleteReq})
	if err != nil {
		return 0, fmt.Errorf("marshal delete query: %w", err)
	}

	url := fmt.Sprintf("%s/indexes/%s/docs/index?api-version=2023-07-01", p.url, p.indexName)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return 0, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("delete by query request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("delete by query failed (%d): %s", resp.StatusCode, string(respBody))
	}

	return 1, nil
}

func (p *azureSearchProvider) Ping(ctx context.Context) error {
	url := fmt.Sprintf("%s/indexes/%s?api-version=2023-07-01", p.url, p.indexName)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("api-key", p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("ping request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("ping failed (%d)", resp.StatusCode)
	}

	return nil
}

func (p *azureSearchProvider) Close() error {
	return nil
}

var _ = strings.Contains
