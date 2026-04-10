package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	return nil, fmt.Errorf("embeddings not implemented for Azure provider")
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
