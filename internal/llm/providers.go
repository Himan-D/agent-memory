package llm

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

type openaiProvider struct {
	apiKey  string
	baseURL string
	org     string
	model   string
}

func newOpenAIProvider(cfg *Config) *openaiProvider {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	return &openaiProvider{
		apiKey:  cfg.APIKey,
		baseURL: baseURL,
		org:     cfg.Organization,
		model:   cfg.OpenAI.Model,
	}
}

func (p *openaiProvider) Name() ProviderType { return ProviderOpenAI }

func (p *openaiProvider) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}

	openaiReq := map[string]interface{}{
		"model":       model,
		"messages":    req.Messages,
		"temperature": req.Temperature,
	}

	if req.MaxTokens > 0 {
		openaiReq["max_tokens"] = req.MaxTokens
	}
	if req.TopP > 0 {
		openaiReq["top_p"] = req.TopP
	}
	if req.FrequencyPenalty != 0 {
		openaiReq["frequency_penalty"] = req.FrequencyPenalty
	}
	if req.PresencePenalty != 0 {
		openaiReq["presence_penalty"] = req.PresencePenalty
	}
	if len(req.Stop) > 0 {
		openaiReq["stop"] = req.Stop
	}

	body, err := json.Marshal(openaiReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	p.setHeaders(httpReq)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error: %s", string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	choices, ok := result["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid choice format")
	}

	content := ""
	if msg, ok := choice["message"].(map[string]interface{}); ok {
		if c, ok := msg["content"].(string); ok {
			content = c
		}
	}

	usage, _ := result["usage"].(map[string]interface{})
	tokens := 0
	if usage != nil {
		if t, ok := usage["total_tokens"].(float64); ok {
			tokens = int(t)
		}
	}

	return &CompletionResponse{
		Content:  content,
		Model:    model,
		Provider: ProviderOpenAI,
		Tokens:   tokens,
	}, nil
}

func (p *openaiProvider) Embed(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	model := req.Model
	if model == "" {
		model = "text-embedding-3-small"
	}

	openaiReq := map[string]interface{}{
		"model": model,
		"input": req.Input,
	}

	body, err := json.Marshal(openaiReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/embeddings", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	p.setHeaders(httpReq)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error: %s", string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	data, ok := result["data"].([]interface{})
	if !ok || len(data) == 0 {
		return nil, fmt.Errorf("no embedding in response")
	}

	embeddingData, ok := data[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid embedding format")
	}

	embedding, ok := embeddingData["embedding"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("no embedding vector")
	}

	floatEmb := make([]float32, len(embedding))
	for i, v := range embedding {
		if f, ok := v.(float64); ok {
			floatEmb[i] = float32(f)
		}
	}

	return &EmbeddingResponse{
		Embedding: floatEmb,
		Model:     model,
		Provider:  ProviderOpenAI,
	}, nil
}

func (p *openaiProvider) Rerank(ctx context.Context, req *RerankRequest) (*RerankResponse, error) {
	return nil, fmt.Errorf("reranking not supported for OpenAI provider")
}

func (p *openaiProvider) setHeaders(r *http.Request) {
	r.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		r.Header.Set("Authorization", "Bearer "+p.apiKey)
	}
	if p.org != "" {
		r.Header.Set("OpenAI-Organization", p.org)
	}
}

type anthropicProvider struct {
	apiKey  string
	baseURL string
	model   string
}

func newAnthropicProvider(cfg *Config) *anthropicProvider {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.anthropic.com/v1"
	}

	return &anthropicProvider{
		apiKey:  cfg.APIKey,
		baseURL: baseURL,
		model:   cfg.Anthropic.Model,
	}
}

func (p *anthropicProvider) Name() ProviderType { return ProviderAnthropic }

func (p *anthropicProvider) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}

	system := ""
	messages := req.Messages

	if len(messages) > 0 && messages[0].Role == "system" {
		system = messages[0].Content
		messages = messages[1:]
	}

	anthropicReq := map[string]interface{}{
		"model":       model,
		"messages":    messages,
		"temperature": req.Temperature,
		"max_tokens":  req.MaxTokens,
	}

	if system != "" {
		anthropicReq["system"] = system
	}

	body, err := json.Marshal(anthropicReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/messages", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error: %s", string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	content := ""
	if c, ok := result["content"].([]interface{}); ok && len(c) > 0 {
		if block, ok := c[0].(map[string]interface{}); ok {
			content, _ = block["text"].(string)
		}
	}

	usage, _ := result["usage"].(map[string]interface{})
	tokens := 0
	if usage != nil {
		if t, ok := usage["output_tokens"].(float64); ok {
			tokens = int(t)
		}
	}

	return &CompletionResponse{
		Content:  content,
		Model:    model,
		Provider: ProviderAnthropic,
		Tokens:   tokens,
	}, nil
}

func (p *anthropicProvider) Embed(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	return nil, fmt.Errorf("embeddings not supported for Anthropic provider")
}

func (p *anthropicProvider) Rerank(ctx context.Context, req *RerankRequest) (*RerankResponse, error) {
	return nil, fmt.Errorf("reranking not supported for Anthropic provider")
}

type azureProvider struct {
	apiKey     string
	endpoint   string
	deployment string
	apiVersion string
}

func newAzureProvider(cfg *Config) *azureProvider {
	endpoint := cfg.Azure.Endpoint
	if endpoint == "" {
		endpoint = "https://your-resource.openai.azure.com"
	}

	return &azureProvider{
		apiKey:     cfg.APIKey,
		endpoint:   endpoint,
		deployment: cfg.Azure.Deployment,
		apiVersion: cfg.Azure.APIVersion,
	}
}

func (p *azureProvider) Name() ProviderType { return ProviderAzure }

func (p *azureProvider) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	model := req.Model
	if model == "" {
		model = p.deployment
	}

	azureReq := map[string]interface{}{
		"messages":    req.Messages,
		"temperature": req.Temperature,
	}

	if req.MaxTokens > 0 {
		azureReq["max_tokens"] = req.MaxTokens
	}

	body, err := json.Marshal(azureReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=%s", p.endpoint, model, p.apiVersion)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("api-key", p.apiKey)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error: %s", string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	choices, ok := result["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	choice := choices[0].(map[string]interface{})
	content := ""
	if msg, ok := choice["message"].(map[string]interface{}); ok {
		content, _ = msg["content"].(string)
	}

	return &CompletionResponse{
		Content:  content,
		Model:    model,
		Provider: ProviderAzure,
	}, nil
}

func (p *azureProvider) Embed(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	deployment := req.Model
	if deployment == "" {
		deployment = p.deployment
	}
	if deployment == "" {
		return nil, fmt.Errorf("Azure embedding deployment name is required (set via model parameter or AZURE_DEPLOYMENT env)")
	}

	azureReq := map[string]interface{}{
		"input": req.Input,
	}

	body, err := json.Marshal(azureReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	embedURL := fmt.Sprintf("%s/openai/deployments/%s/embeddings?api-version=%s", p.endpoint, deployment, p.apiVersion)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", embedURL, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("api-key", p.apiKey)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("Azure API error: %s", string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	data, ok := result["data"].([]interface{})
	if !ok || len(data) == 0 {
		return nil, fmt.Errorf("no embedding in response")
	}

	embeddingData, ok := data[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid embedding format")
	}

	embedding, ok := embeddingData["embedding"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("no embedding vector")
	}

	floatEmb := make([]float32, len(embedding))
	for i, v := range embedding {
		if f, ok := v.(float64); ok {
			floatEmb[i] = float32(f)
		}
	}

	return &EmbeddingResponse{
		Embedding: floatEmb,
		Model:     deployment,
		Provider:  ProviderAzure,
	}, nil
}

func (p *azureProvider) Rerank(ctx context.Context, req *RerankRequest) (*RerankResponse, error) {
	return nil, fmt.Errorf("reranking not implemented for Azure provider")
}

type googleProvider struct {
	apiKey  string
	baseURL string
	model   string
}

func newGoogleProvider(cfg *Config) *googleProvider {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com/v1beta"
	}

	return &googleProvider{
		apiKey:  cfg.APIKey,
		baseURL: baseURL,
		model:   cfg.Google.Model,
	}
}

func (p *googleProvider) Name() ProviderType { return ProviderGoogle }

func (p *googleProvider) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}

	contents := make([]map[string]interface{}, 0)
	for _, msg := range req.Messages {
		role := msg.Role
		if role == "assistant" {
			role = "model"
		}
		contents = append(contents, map[string]interface{}{
			"role":  role,
			"parts": []map[string]string{{"text": msg.Content}},
		})
	}

	googleReq := map[string]interface{}{
		"contents": contents,
		"generationConfig": map[string]interface{}{
			"temperature":     req.Temperature,
			"maxOutputTokens": req.MaxTokens,
		},
	}

	body, err := json.Marshal(googleReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", p.baseURL, model, p.apiKey)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error: %s", string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	content := ""
	if candidates, ok := result["candidates"].([]interface{}); ok && len(candidates) > 0 {
		if cand, ok := candidates[0].(map[string]interface{}); ok {
			if contentB, ok := cand["content"].(map[string]interface{}); ok {
				if parts, ok := contentB["parts"].([]interface{}); ok && len(parts) > 0 {
					if part, ok := parts[0].(map[string]interface{}); ok {
						content, _ = part["text"].(string)
					}
				}
			}
		}
	}

	return &CompletionResponse{
		Content:  content,
		Model:    model,
		Provider: ProviderGoogle,
	}, nil
}

func (p *googleProvider) Embed(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	model := req.Model
	if model == "" {
		model = "text-embedding-004"
	}

	googleReq := map[string]interface{}{
		"model": model,
		"text":  req.Input,
	}

	body, err := json.Marshal(googleReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/models/%s:embedContent?key=%s", p.baseURL, model, p.apiKey)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error: %s", string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	embedding := make([]float32, 768)
	if emb, ok := result["embedding"].(map[string]interface{}); ok {
		if values, ok := emb["values"].([]interface{}); ok {
			for i, v := range values {
				if f, ok := v.(float64); ok && i < len(embedding) {
					embedding[i] = float32(f)
				}
			}
		}
	}

	return &EmbeddingResponse{
		Embedding: embedding,
		Model:     model,
		Provider:  ProviderGoogle,
	}, nil
}

func (p *googleProvider) Rerank(ctx context.Context, req *RerankRequest) (*RerankResponse, error) {
	return nil, fmt.Errorf("reranking not supported for Google provider")
}

type mistralProvider struct {
	apiKey  string
	baseURL string
	model   string
}

func newMistralProvider(cfg *Config) *mistralProvider {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.mistral.ai/v1"
	}

	return &mistralProvider{
		apiKey:  cfg.APIKey,
		baseURL: baseURL,
		model:   cfg.Mistral.Model,
	}
}

func (p *mistralProvider) Name() ProviderType { return ProviderMistral }

func (p *mistralProvider) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}

	mistralReq := map[string]interface{}{
		"model":       model,
		"messages":    req.Messages,
		"temperature": req.Temperature,
	}

	if req.MaxTokens > 0 {
		mistralReq["max_tokens"] = req.MaxTokens
	}

	body, err := json.Marshal(mistralReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error: %s", string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	choices, ok := result["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	choice := choices[0].(map[string]interface{})
	content := ""
	if msg, ok := choice["message"].(map[string]interface{}); ok {
		content, _ = msg["content"].(string)
	}

	return &CompletionResponse{
		Content:  content,
		Model:    model,
		Provider: ProviderMistral,
	}, nil
}

func (p *mistralProvider) Embed(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	model := req.Model
	if model == "" {
		model = "mistral-embed"
	}

	mistralReq := map[string]interface{}{
		"model": model,
		"input": req.Input,
	}

	body, err := json.Marshal(mistralReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/embeddings", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error: %s", string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	data, ok := result["data"].([]interface{})
	if !ok || len(data) == 0 {
		return nil, fmt.Errorf("no embedding in response")
	}

	embeddingData := data[0].(map[string]interface{})
	embedding := make([]float32, 1024)
	if emb, ok := embeddingData["embedding"].([]interface{}); ok {
		for i, v := range emb {
			if f, ok := v.(float64); ok && i < len(embedding) {
				embedding[i] = float32(f)
			}
		}
	}

	return &EmbeddingResponse{
		Embedding: embedding,
		Model:     model,
		Provider:  ProviderMistral,
	}, nil
}

func (p *mistralProvider) Rerank(ctx context.Context, req *RerankRequest) (*RerankResponse, error) {
	return nil, fmt.Errorf("reranking not supported for Mistral provider")
}

type cohereProvider struct {
	apiKey  string
	baseURL string
	model   string
}

func newCohereProvider(cfg *Config) *cohereProvider {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.cohere.ai/v2"
	}

	return &cohereProvider{
		apiKey:  cfg.APIKey,
		baseURL: baseURL,
		model:   cfg.Cohere.Model,
	}
}

func (p *cohereProvider) Name() ProviderType { return ProviderCohere }

func (p *cohereProvider) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}

	cohereReq := map[string]interface{}{
		"model":       model,
		"messages":    req.Messages,
		"temperature": req.Temperature,
	}

	if req.MaxTokens > 0 {
		cohereReq["max_tokens"] = req.MaxTokens
	}

	body, err := json.Marshal(cohereReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error: %s", string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	content := ""
	if msg, ok := result["text"].(string); ok {
		content = msg
	}

	return &CompletionResponse{
		Content:  content,
		Model:    model,
		Provider: ProviderCohere,
	}, nil
}

func (p *cohereProvider) Embed(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	model := req.Model
	if model == "" {
		model = "embed-english-v3.0"
	}

	cohereReq := map[string]interface{}{
		"model": model,
		"texts": []string{req.Input},
	}

	body, err := json.Marshal(cohereReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/embed", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error: %s", string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	embeddings, ok := result["embeddings"].([]interface{})
	if !ok || len(embeddings) == 0 {
		return nil, fmt.Errorf("no embedding in response")
	}

	emb := embeddings[0].([]interface{})
	floatEmb := make([]float32, len(emb))
	for i, v := range emb {
		if f, ok := v.(float64); ok {
			floatEmb[i] = float32(f)
		}
	}

	return &EmbeddingResponse{
		Embedding: floatEmb,
		Model:     model,
		Provider:  ProviderCohere,
	}, nil
}

func (p *cohereProvider) Rerank(ctx context.Context, req *RerankRequest) (*RerankResponse, error) {
	model := req.Model
	if model == "" {
		model = "rerank-english-v3.0"
	}

	cohereReq := map[string]interface{}{
		"model":     model,
		"query":     req.Query,
		"documents": req.Documents,
		"top_n":     req.TopK,
	}

	body, err := json.Marshal(cohereReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/rerank", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error: %s", string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	results := []RerankResult{}
	if resultsData, ok := result["results"].([]interface{}); ok {
		for _, r := range resultsData {
			if rmap, ok := r.(map[string]interface{}); ok {
				results = append(results, RerankResult{
					Index:    int(rmap["index"].(float64)),
					Document: req.Documents[int(rmap["index"].(float64))],
					Score:    rmap["relevance_score"].(float64),
				})
			}
		}
	}

	return &RerankResponse{
		Results:  results,
		Model:    model,
		Provider: ProviderCohere,
	}, nil
}

type localProvider struct {
	url   string
	model string
}

func newLocalProvider(cfg *Config) *localProvider {
	url := cfg.Local.URL
	if url == "" {
		url = "http://localhost:11434"
	}

	return &localProvider{
		url:   url,
		model: cfg.Local.Model,
	}
}

func (p *localProvider) Name() ProviderType { return ProviderLocal }

func (p *localProvider) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}

	messages := make([]map[string]string, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = map[string]string{
			"role":    msg.Role,
			"content": msg.Content,
		}
	}

	ollamaReq := map[string]interface{}{
		"model":    model,
		"messages": messages,
		"stream":   false,
	}

	if req.Temperature > 0 {
		ollamaReq["temperature"] = req.Temperature
	}
	if req.MaxTokens > 0 {
		ollamaReq["options"] = map[string]int{"num_predict": req.MaxTokens}
	}

	body, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.url+"/api/chat", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error: %s", string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	content := ""
	if msg, ok := result["message"].(map[string]interface{}); ok {
		content, _ = msg["content"].(string)
	}

	return &CompletionResponse{
		Content:  content,
		Model:    model,
		Provider: ProviderLocal,
	}, nil
}

func (p *localProvider) Embed(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	model := req.Model
	if model == "" {
		model = p.model + "-embed"
	}

	ollamaReq := map[string]interface{}{
		"model": model,
		"input": req.Input,
	}

	body, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.url+"/api/embeddings", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error: %s", string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	embedding := make([]float32, 4096)
	if emb, ok := result["embedding"].([]interface{}); ok {
		for i, v := range emb {
			if f, ok := v.(float64); ok && i < len(embedding) {
				embedding[i] = float32(f)
			}
		}
	}

	return &EmbeddingResponse{
		Embedding: embedding,
		Model:     model,
		Provider:  ProviderLocal,
	}, nil
}

func (p *localProvider) Rerank(ctx context.Context, req *RerankRequest) (*RerankResponse, error) {
	return nil, fmt.Errorf("reranking not supported for local provider")
}

type awsProvider struct {
	region          string
	model           string
	embedModel      string
	temperature     float64
	maxTokens       int
	accessKeyID     string
	secretAccessKey string
	sessionToken    string
}

func newAWSProvider(cfg *Config) *awsProvider {
	return &awsProvider{
		region:          cfg.AWS.Region,
		model:           cfg.AWS.Model,
		embedModel:      cfg.AWS.EmbedModel,
		temperature:     cfg.AWS.Temperature,
		maxTokens:       cfg.AWS.MaxTokens,
		accessKeyID:     cfg.AWS.AccessKeyID,
		secretAccessKey: cfg.AWS.SecretAccessKey,
	}
}

func (p *awsProvider) Name() ProviderType { return ProviderAWS }

func (p *awsProvider) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}

	awsReq := map[string]interface{}{
		"model":       model,
		"messages":    req.Messages,
		"temperature": req.Temperature,
	}

	if req.MaxTokens > 0 {
		awsReq["max_tokens"] = req.MaxTokens
	}

	body, err := json.Marshal(awsReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	endpoint := fmt.Sprintf("https://bedrock-runtime.%s.amazonaws.com/model/%s/invoke", p.region, model)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	now := time.Now().UTC()
	amzDate := now.Format("20060102T150405Z")
	dateStamp := now.Format("20060102")
	host := fmt.Sprintf("bedrock-runtime.%s.amazonaws.com", p.region)
	httpReq.Header.Set("Host", host)
	httpReq.Header.Set("X-Amz-Date", amzDate)
	if p.sessionToken != "" {
		httpReq.Header.Set("X-Amz-Security-Token", p.sessionToken)
	}

	parsedURL, _ := url.Parse(endpoint)
	canonicalURI := parsedURL.Path
	canonicalQueryString := parsedURL.RawQuery

	var signedHeaders []string
	for k := range httpReq.Header {
		signedHeaders = append(signedHeaders, strings.ToLower(k))
	}
	sort.Strings(signedHeaders)

	var canonicalHeaderLines []string
	var signedHeaderNames []string
	for _, k := range signedHeaders {
		val := strings.TrimSpace(httpReq.Header.Get(k))
		canonicalHeaderLines = append(canonicalHeaderLines, fmt.Sprintf("%s:%s", k, val))
		signedHeaderNames = append(signedHeaderNames, k)
	}
	signedHeadersStr := strings.Join(signedHeaderNames, ";")
	canonicalHeaders := strings.Join(canonicalHeaderLines, "\n") + "\n"

	bodyHash := sha256.Sum256(body)
	payloadHash := hex.EncodeToString(bodyHash[:])

	canonicalRequest := strings.Join([]string{
		"POST",
		canonicalURI,
		canonicalQueryString,
		canonicalHeaders,
		signedHeadersStr,
		payloadHash,
	}, "\n")

	canonicalRequestHash := sha256.Sum256([]byte(canonicalRequest))
	credentialScope := fmt.Sprintf("%s/%s/aws4_request", dateStamp, "bedrock")
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		credentialScope,
		hex.EncodeToString(canonicalRequestHash[:]),
	}, "\n")

	signingKey := deriveSigningKey(p.secretAccessKey, dateStamp, "bedrock", p.region)
	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

	authHeader := fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		p.accessKeyID, credentialScope, signedHeadersStr, signature)
	httpReq.Header.Set("Authorization", authHeader)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("AWS Bedrock API error: %s", string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	content := ""
	if msg, ok := result["message"].(map[string]interface{}); ok {
		if c, ok := msg["content"].(string); ok {
			content = c
		}
	}

	return &CompletionResponse{
		Content:  content,
		Model:    model,
		Provider: ProviderAWS,
	}, nil
}

func (p *awsProvider) Embed(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	return nil, fmt.Errorf("embeddings not supported for AWS Bedrock provider")
}

func (p *awsProvider) Rerank(ctx context.Context, req *RerankRequest) (*RerankResponse, error) {
	return nil, fmt.Errorf("reranking not supported for AWS Bedrock provider")
}

type groqProvider struct {
	apiKey  string
	baseURL string
	model   string
}

func newGroqProvider(cfg *Config) *groqProvider {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.groq.com/openai/v1"
	}

	return &groqProvider{
		apiKey:  cfg.APIKey,
		baseURL: baseURL,
		model:   cfg.Groq.Model,
	}
}

func (p *groqProvider) Name() ProviderType { return ProviderGroq }

func (p *groqProvider) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}

	groqReq := map[string]interface{}{
		"model":       model,
		"messages":    req.Messages,
		"temperature": req.Temperature,
	}

	if req.MaxTokens > 0 {
		groqReq["max_tokens"] = req.MaxTokens
	}

	body, err := json.Marshal(groqReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("Groq API error: %s", string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	choices, ok := result["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	choice := choices[0].(map[string]interface{})
	content := ""
	if msg, ok := choice["message"].(map[string]interface{}); ok {
		content, _ = msg["content"].(string)
	}

	return &CompletionResponse{
		Content:  content,
		Model:    model,
		Provider: ProviderGroq,
	}, nil
}

func (p *groqProvider) Embed(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	model := req.Model
	if model == "" {
		return nil, fmt.Errorf("Groq does not support embeddings natively; configure a compatible embedding provider (e.g., OpenAI, LiteLLM)")
	}

	groqReq := map[string]interface{}{
		"model": model,
		"input": req.Input,
	}

	body, err := json.Marshal(groqReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/embeddings", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("Groq API error: %s", string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	data, ok := result["data"].([]interface{})
	if !ok || len(data) == 0 {
		return nil, fmt.Errorf("no embedding in response")
	}

	embeddingData := data[0].(map[string]interface{})
	embedding := make([]float32, 1024)
	if emb, ok := embeddingData["embedding"].([]interface{}); ok {
		for i, v := range emb {
			if f, ok := v.(float64); ok && i < len(embedding) {
				embedding[i] = float32(f)
			}
		}
	}

	return &EmbeddingResponse{
		Embedding: embedding,
		Model:     model,
		Provider:  ProviderGroq,
	}, nil
}

func (p *groqProvider) Rerank(ctx context.Context, req *RerankRequest) (*RerankResponse, error) {
	return nil, fmt.Errorf("reranking not supported for Groq provider")
}

type deepseekProvider struct {
	apiKey  string
	baseURL string
	model   string
}

func newDeepSeekProvider(cfg *Config) *deepseekProvider {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.deepseek.com"
	}

	return &deepseekProvider{
		apiKey:  cfg.APIKey,
		baseURL: baseURL,
		model:   cfg.DeepSeek.Model,
	}
}

func (p *deepseekProvider) Name() ProviderType { return ProviderDeepSeek }

func (p *deepseekProvider) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}

	messages := make([]map[string]interface{}, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		}
	}

	deepseekReq := map[string]interface{}{
		"model":       model,
		"messages":    messages,
		"temperature": req.Temperature,
	}

	if req.MaxTokens > 0 {
		deepseekReq["max_tokens"] = req.MaxTokens
	}

	body, err := json.Marshal(deepseekReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("DeepSeek API error: %s", string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	choices, ok := result["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	choice := choices[0].(map[string]interface{})
	content := ""
	if msg, ok := choice["message"].(map[string]interface{}); ok {
		content, _ = msg["content"].(string)
	}

	usage, _ := result["usage"].(map[string]interface{})
	tokens := 0
	if usage != nil {
		if t, ok := usage["total_tokens"].(float64); ok {
			tokens = int(t)
		}
	}

	return &CompletionResponse{
		Content:  content,
		Model:    model,
		Provider: ProviderDeepSeek,
		Tokens:   tokens,
	}, nil
}

func (p *deepseekProvider) Embed(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	model := req.Model
	if model == "" {
		return nil, fmt.Errorf("DeepSeek does not have a public embedding model; set an explicit model name or use a different embedding provider")
	}

	deepseekReq := map[string]interface{}{
		"model": model,
		"input": req.Input,
	}

	body, err := json.Marshal(deepseekReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/embeddings", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("DeepSeek API error: %s", string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	data, ok := result["data"].([]interface{})
	if !ok || len(data) == 0 {
		return nil, fmt.Errorf("no embedding in response")
	}

	embeddingData := data[0].(map[string]interface{})
	embedding := make([]float32, 1024)
	if emb, ok := embeddingData["embedding"].([]interface{}); ok {
		for i, v := range emb {
			if f, ok := v.(float64); ok && i < len(embedding) {
				embedding[i] = float32(f)
			}
		}
	}

	return &EmbeddingResponse{
		Embedding: embedding,
		Model:     model,
		Provider:  ProviderDeepSeek,
	}, nil
}

func (p *deepseekProvider) Rerank(ctx context.Context, req *RerankRequest) (*RerankResponse, error) {
	return nil, fmt.Errorf("reranking not supported for DeepSeek provider")
}

// litellmProvider wraps the OpenAI-compatible LiteLLM proxy.
// LiteLLM (https://github.com/BerriAI/litellm) provides a single OpenAI-format
// endpoint that routes to 100+ providers. Set LITELLM_BASE_URL to point at your
// proxy (default http://localhost:4000).
type litellmProvider struct {
	apiKey  string
	baseURL string
	model   string
}

func newLiteLLMProvider(cfg *Config) *litellmProvider {
	baseURL := cfg.LiteLLM.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:4000"
	}
	model := cfg.LiteLLM.Model
	if model == "" {
		model = "gpt-4o-mini"
	}
	return &litellmProvider{
		apiKey:  cfg.APIKey,
		baseURL: baseURL,
		model:   model,
	}
}

func (p *litellmProvider) Name() ProviderType { return ProviderLiteLLM }

// Complete sends a chat completion request to the LiteLLM proxy using the
// OpenAI-compatible /chat/completions endpoint.
func (p *litellmProvider) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	// Delegate to the OpenAI-compatible implementation by constructing a
	// temporary openaiProvider with the LiteLLM base URL.
	delegate := &openaiProvider{
		apiKey:  p.apiKey,
		baseURL: p.baseURL,
		model:   p.model,
	}
	resp, err := delegate.Complete(ctx, req)
	if err != nil {
		return nil, err
	}
	// Override provider label so callers know which proxy was used.
	resp.Provider = ProviderLiteLLM
	return resp, nil
}

// Embed sends an embedding request to the LiteLLM proxy using the
// OpenAI-compatible /embeddings endpoint.
func (p *litellmProvider) Embed(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	delegate := &openaiProvider{
		apiKey:  p.apiKey,
		baseURL: p.baseURL,
		model:   p.model,
	}
	resp, err := delegate.Embed(ctx, req)
	if err != nil {
		return nil, err
	}
	resp.Provider = ProviderLiteLLM
	return resp, nil
}

func (p *litellmProvider) Rerank(ctx context.Context, req *RerankRequest) (*RerankResponse, error) {
	return nil, fmt.Errorf("reranking not supported for LiteLLM provider")
}

func hmacSHA256(key []byte, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func deriveSigningKey(secretKey, dateStamp, service, region string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secretKey), []byte(dateStamp))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte(service))
	kSigning := hmacSHA256(kService, []byte("aws4_request"))
	return kSigning
}
