package embedding

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"agent-memory/internal/config"
)

type OpenAIEmbedding struct {
	config config.OpenAIConfig
	client *http.Client
}

type embedRequest struct {
	Input string `json:"input"`
	Model string `json:"model"`
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

func NewOpenAI(cfg config.OpenAIConfig) *OpenAIEmbedding {
	return &OpenAIEmbedding{
		config: cfg,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (e *OpenAIEmbedding) GenerateEmbedding(text string) ([]float32, error) {
	if e.config.APIKey == "" {
		return nil, fmt.Errorf("openai API key not configured")
	}

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
	if e.config.APIKey == "" {
		return nil, fmt.Errorf("openai API key not configured")
	}

	var embeddings [][]float32

	// Process in batches of 100
	for i := 0; i < len(texts); i += 100 {
		end := i + 100
		if end > len(texts) {
			end = len(texts)
		}
		batch := texts[i:end]

		batchEmbeddings, err := e.generateBatch(batch)
		if err != nil {
			return nil, fmt.Errorf("batch %d-%d: %w", i, end, err)
		}
		embeddings = append(embeddings, batchEmbeddings...)
	}

	return embeddings, nil
}

func (e *OpenAIEmbedding) generateBatch(texts []string) ([][]float32, error) {
	reqBody := struct {
		Input []string `json:"input"`
		Model string   `json:"model"`
	}{
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
