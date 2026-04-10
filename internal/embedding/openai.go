package embedding

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"agent-memory/internal/config"
)

type OpenAIEmbedding struct {
	config    config.OpenAIConfig
	client    *http.Client
	cache     *EmbeddingCache
	rateLimit *rateLimiter
}

type embedRequest struct {
	Input string `json:"input"`
	Model string `json:"model"`
}

type embedBatchRequest struct {
	Input []string `json:"input"`
	Model string   `json:"model"`
}

type embedResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

type EmbeddingCache struct {
	mu        sync.RWMutex
	entries   map[string]*CacheEntry
	lru       []string
	capacity  int
	hitCount  int64
	missCount int64
}

type CacheEntry struct {
	Embedding []float32
	TextHash  string
	CreatedAt time.Time
	Accessed  time.Time
}

func NewEmbeddingCache(capacity int) *EmbeddingCache {
	return &EmbeddingCache{
		entries:  make(map[string]*CacheEntry),
		lru:      make([]string, 0, capacity),
		capacity: capacity,
	}
}

func (c *EmbeddingCache) Get(text string) ([]float32, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	hash := c.hashText(text)
	entry, exists := c.entries[hash]
	if !exists {
		c.missCount++
		return nil, false
	}

	c.hitCount++
	entry.Accessed = time.Now()
	c.moveToFront(hash)
	return entry.Embedding, true
}

func (c *EmbeddingCache) Set(text string, embedding []float32) {
	c.mu.Lock()
	defer c.mu.Unlock()

	hash := c.hashText(text)
	if _, exists := c.entries[hash]; exists {
		c.entries[hash].Embedding = embedding
		c.entries[hash].Accessed = time.Now()
		c.moveToFront(hash)
		return
	}

	if len(c.entries) >= c.capacity {
		c.evictOldest()
	}

	c.entries[hash] = &CacheEntry{
		Embedding: embedding,
		TextHash:  hash,
		CreatedAt: time.Now(),
		Accessed:  time.Now(),
	}
	c.lru = append([]string{hash}, c.lru...)
}

func (c *EmbeddingCache) hashText(text string) string {
	hash := sha256.Sum256([]byte(text))
	return hex.EncodeToString(hash[:])
}

func (c *EmbeddingCache) moveToFront(hash string) {
	for i, h := range c.lru {
		if h == hash {
			c.lru = append([]string{hash}, append(c.lru[:i], c.lru[i+1:]...)...)
			return
		}
	}
}

func (c *EmbeddingCache) evictOldest() {
	if len(c.lru) == 0 {
		return
	}
	oldest := c.lru[len(c.lru)-1]
	delete(c.entries, oldest)
	c.lru = c.lru[:len(c.lru)-1]
}

func (c *EmbeddingCache) Stats() (hits, misses int64, size int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.hitCount, c.missCount, len(c.entries)
}

func (c *EmbeddingCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]*CacheEntry)
	c.lru = make([]string, 0, c.capacity)
	c.hitCount = 0
	c.missCount = 0
}

type rateLimiter struct {
	mu         sync.Mutex
	tokens     float64
	maxTokens  float64
	refillRate float64
	lastRefill time.Time
}

func newRateLimiter(maxTokens, refillRate float64) *rateLimiter {
	return &rateLimiter{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

func (r *rateLimiter) Allow(count int) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.refill()

	if r.tokens >= float64(count) {
		r.tokens -= float64(count)
		return true
	}
	return false
}

func (r *rateLimiter) refill() {
	now := time.Now()
	elapsed := now.Sub(r.lastRefill).Seconds()
	r.tokens = min(r.maxTokens, r.tokens+(elapsed*r.refillRate))
	r.lastRefill = now
}

func (r *rateLimiter) WaitTime() time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.refill()
	if r.tokens >= 1 {
		return 0
	}
	return time.Duration((1 - r.tokens) / r.refillRate * float64(time.Second))
}

func NewOpenAI(cfg config.OpenAIConfig) *OpenAIEmbedding {
	return &OpenAIEmbedding{
		config:    cfg,
		client:    &http.Client{Timeout: 60 * time.Second},
		cache:     NewEmbeddingCache(10000),
		rateLimit: newRateLimiter(500, 500),
	}
}

func (e *OpenAIEmbedding) GenerateEmbedding(text string) ([]float32, error) {
	return e.GenerateEmbeddingWithContext(context.Background(), text)
}

func (e *OpenAIEmbedding) GenerateEmbeddingWithContext(ctx context.Context, text string) ([]float32, error) {
	if e.config.APIKey == "" {
		return nil, fmt.Errorf("openai API key not configured")
	}

	if emb, found := e.cache.Get(text); found {
		embCopy := make([]float32, len(emb))
		copy(embCopy, emb)
		return embCopy, nil
	}

	for !e.rateLimit.Allow(1) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(e.rateLimit.WaitTime()):
		}
	}

	emb, err := e.generateEmbeddingRequest(text)
	if err != nil {
		return nil, err
	}

	e.cache.Set(text, emb)
	return emb, nil
}

func (e *OpenAIEmbedding) generateEmbeddingRequest(text string) ([]float32, error) {
	reqBody := embedRequest{
		Input: text,
		Model: e.config.Model,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/embeddings", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.config.APIKey)

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openai API error (%d): %s", resp.StatusCode, string(body))
	}

	var embedResp embedResponse
	if err := json.Unmarshal(body, &embedResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if len(embedResp.Data) == 0 {
		return nil, fmt.Errorf("no embedding data returned")
	}

	return embedResp.Data[0].Embedding, nil
}

func (e *OpenAIEmbedding) GenerateBatchEmbeddings(texts []string) ([][]float32, error) {
	return e.GenerateBatchEmbeddingsWithContext(context.Background(), texts)
}

func (e *OpenAIEmbedding) GenerateBatchEmbeddingsWithContext(ctx context.Context, texts []string) ([][]float32, error) {
	if e.config.APIKey == "" {
		return nil, fmt.Errorf("openai API key not configured")
	}

	var results [][]float32
	var textsToFetch []string
	var indices []int

	for i, text := range texts {
		if emb, found := e.cache.Get(text); found {
			embCopy := make([]float32, len(emb))
			copy(embCopy, emb)
			results = append(results, embCopy)
		} else {
			textsToFetch = append(textsToFetch, text)
			indices = append(indices, i)
		}
	}

	if len(textsToFetch) == 0 {
		return results, nil
	}

	for i := 0; i < len(textsToFetch); i += 100 {
		end := i + 100
		if end > len(textsToFetch) {
			end = len(textsToFetch)
		}
		batch := textsToFetch[i:end]

		for !e.rateLimit.Allow(len(batch)) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(e.rateLimit.WaitTime()):
			}
		}

		embeddings, err := e.generateBatch(batch)
		if err != nil {
			return nil, fmt.Errorf("batch %d-%d: %w", i, end, err)
		}

		for j, emb := range embeddings {
			originalIdx := indices[i+j]
			e.cache.Set(textsToFetch[i+j], emb)

			for len(results) <= originalIdx {
				results = append(results, nil)
			}
			results[originalIdx] = emb
		}
	}

	result := make([][]float32, len(texts))
	copy(result, results)
	return result, nil
}

func (e *OpenAIEmbedding) generateBatch(texts []string) ([][]float32, error) {
	reqBody := embedBatchRequest{
		Input: texts,
		Model: e.config.Model,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/embeddings", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.config.APIKey)

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openai API error (%d): %s", resp.StatusCode, string(body))
	}

	var embedResp embedResponse
	if err := json.Unmarshal(body, &embedResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	embeddings := make([][]float32, len(embedResp.Data))
	for _, d := range embedResp.Data {
		embeddings[d.Index] = d.Embedding
	}

	return embeddings, nil
}

func (e *OpenAIEmbedding) CacheStats() (hits, misses int64, size int) {
	return e.cache.Stats()
}

func (e *OpenAIEmbedding) ClearCache() {
	e.cache.Clear()
}

func (e *OpenAIEmbedding) PreloadEmbeddings(texts []string) error {
	_, err := e.GenerateBatchEmbeddings(texts)
	return err
}
