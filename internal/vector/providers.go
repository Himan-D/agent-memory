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
}

func newWeaviateProvider(cfg *Config) (*weaviateProvider, error) {
	return &weaviateProvider{
		url:       cfg.Weaviate.URL,
		apiKey:    cfg.Weaviate.APIKey,
		className: cfg.Weaviate.ClassName,
	}, nil
}

func (p *weaviateProvider) Name() ProviderType { return ProviderWeaviate }

func (p *weaviateProvider) StoreEmbedding(ctx context.Context, text string, id string, embedding []float32, meta map[string]interface{}) (string, error) {
	return id, nil
}

func (p *weaviateProvider) Search(ctx context.Context, query []float32, limit int, threshold float32, filters map[string]interface{}) ([]types.MemoryResult, error) {
	return []types.MemoryResult{}, nil
}

func (p *weaviateProvider) UpdateMemory(ctx context.Context, id string, text string, meta map[string]interface{}) error {
	return nil
}

func (p *weaviateProvider) DeleteMemory(ctx context.Context, id string) error {
	return nil
}

func (p *weaviateProvider) UpdateVector(ctx context.Context, id string, embedding []float32) error {
	return nil
}

func (p *weaviateProvider) DeleteByFilter(ctx context.Context, filter map[string]interface{}) (int, error) {
	return 0, nil
}

func (p *weaviateProvider) Ping(ctx context.Context) error {
	return nil
}

func (p *weaviateProvider) Close() error {
	return nil
}

type chromaProvider struct {
	url        string
	apiKey     string
	collection string
}

func newChromaProvider(cfg *Config) (*chromaProvider, error) {
	return &chromaProvider{
		url:        cfg.Chroma.URL,
		apiKey:     cfg.Chroma.APIKey,
		collection: cfg.Chroma.Collection,
	}, nil
}

func (p *chromaProvider) Name() ProviderType { return ProviderChroma }

func (p *chromaProvider) StoreEmbedding(ctx context.Context, text string, id string, embedding []float32, meta map[string]interface{}) (string, error) {
	return id, nil
}

func (p *chromaProvider) Search(ctx context.Context, query []float32, limit int, threshold float32, filters map[string]interface{}) ([]types.MemoryResult, error) {
	return []types.MemoryResult{}, nil
}

func (p *chromaProvider) UpdateMemory(ctx context.Context, id string, text string, meta map[string]interface{}) error {
	return nil
}

func (p *chromaProvider) DeleteMemory(ctx context.Context, id string) error {
	return nil
}

func (p *chromaProvider) UpdateVector(ctx context.Context, id string, embedding []float32) error {
	return nil
}

func (p *chromaProvider) DeleteByFilter(ctx context.Context, filter map[string]interface{}) (int, error) {
	return 0, nil
}

func (p *chromaProvider) Ping(ctx context.Context) error {
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
}

func newElasticProvider(cfg *Config) (*elasticProvider, error) {
	return &elasticProvider{
		addresses: cfg.Elastic.Addresses,
		apiKey:    cfg.Elastic.APIKey,
		index:     cfg.Elastic.Index,
	}, nil
}

func (p *elasticProvider) Name() ProviderType { return ProviderElastic }

func (p *elasticProvider) StoreEmbedding(ctx context.Context, text string, id string, embedding []float32, meta map[string]interface{}) (string, error) {
	return id, nil
}

func (p *elasticProvider) Search(ctx context.Context, query []float32, limit int, threshold float32, filters map[string]interface{}) ([]types.MemoryResult, error) {
	return []types.MemoryResult{}, nil
}

func (p *elasticProvider) UpdateMemory(ctx context.Context, id string, text string, meta map[string]interface{}) error {
	return nil
}

func (p *elasticProvider) DeleteMemory(ctx context.Context, id string) error {
	return nil
}

func (p *elasticProvider) UpdateVector(ctx context.Context, id string, embedding []float32) error {
	return nil
}

func (p *elasticProvider) DeleteByFilter(ctx context.Context, filter map[string]interface{}) (int, error) {
	return 0, nil
}

func (p *elasticProvider) Ping(ctx context.Context) error {
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

var _ = strings.Contains
