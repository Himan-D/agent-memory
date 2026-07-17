package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"agent-memory/internal/memory/types"
)

// OpenAI-compatible request/response types

type ChatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Stream      bool          `json:"stream,omitempty"`
	Temperature *float64      `json:"temperature,omitempty"`
	MaxTokens   *int          `json:"max_tokens,omitempty"`
	TopP        *float64      `json:"top_p,omitempty"`
	User        string        `json:"user,omitempty"`
	// Hystersis extensions
	MemoryEnabled   *bool  `json:"memory_enabled,omitempty"` // default true
	MemoryUserID    string `json:"memory_user_id,omitempty"`
	MemoryAgentID   string `json:"memory_agent_id,omitempty"`
	MemorySessionID string `json:"memory_session_id,omitempty"`
	MemoryMaxTokens int    `json:"memory_max_tokens,omitempty"` // token budget for memories
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
	// Hystersis extension
	MemoriesUsed int `json:"memories_used,omitempty"`
}

type Choice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

func (s *APIServer) openAIChatCompletionsHandler(w http.ResponseWriter, r *http.Request) {
	var req ChatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":{"message":"Invalid request body","type":"invalid_request_error"}}`, http.StatusBadRequest)
		return
	}

	// Memory injection (enabled by default)
	memoryEnabled := req.MemoryEnabled == nil || *req.MemoryEnabled
	var memoriesUsed int

	if memoryEnabled && s.memSvc != nil {
		// Extract the last user message as the search query
		var query string
		for i := len(req.Messages) - 1; i >= 0; i-- {
			if req.Messages[i].Role == "user" {
				query = req.Messages[i].Content
				break
			}
		}

		if query != "" {
			ctx := context.Background()
			userID := req.MemoryUserID
			if userID == "" {
				userID = req.User
			}

			maxTokens := req.MemoryMaxTokens
			if maxTokens <= 0 {
				maxTokens = 3000 // Default memory budget
			}

			searchReq := &types.SearchRequest{
				Query:   query,
				UserID:  userID,
				AgentID: req.MemoryAgentID,
				Limit:   10,
			}

			searchResults, err := s.memSvc.SearchMemories(ctx, searchReq)
			if err == nil && len(searchResults) > 0 {
				// Build memory context with token budget
				var memoryLines []string
				tokenCount := 0
				for _, mem := range searchResults {
					text := mem.Text
					if text == "" && mem.Metadata != nil {
						text = mem.Metadata.Content
					}
					if text == "" {
						continue
					}
					memTokens := len(text) / 4
					if tokenCount+memTokens > maxTokens {
						break
					}
					memoryLines = append(memoryLines, fmt.Sprintf("- %s", text))
					tokenCount += memTokens
					memoriesUsed++
				}

				if len(memoryLines) > 0 {
					memoryContext := fmt.Sprintf(
						"## Relevant Memories\nThe following are relevant memories about this user/context:\n%s\n\nUse these memories to personalize your response when relevant.",
						strings.Join(memoryLines, "\n"),
					)

					// Inject as first system message or prepend to existing
					injected := false
					for i, msg := range req.Messages {
						if msg.Role == "system" {
							req.Messages[i].Content = msg.Content + "\n\n" + memoryContext
							injected = true
							break
						}
					}
					if !injected {
						req.Messages = append([]ChatMessage{{Role: "system", Content: memoryContext}}, req.Messages...)
					}
				}
			}

			// Store the user message as a new memory (async, best-effort)
			if userID != "" {
				go func() {
					bgCtx := context.Background()
					_, _ = s.memSvc.CreateMemory(bgCtx, &types.Memory{
						Content:   query,
						UserID:    userID,
						AgentID:   req.MemoryAgentID,
						SessionID: req.MemorySessionID,
						Type:      types.MemoryTypeConversation,
					})
				}()
			}
		}
	}

	// Resolve upstream LLM base URL and API key
	llmBaseURL := s.cfg.LLM.BaseURL
	if llmBaseURL == "" {
		llmBaseURL = "https://api.openai.com"
	}
	llmAPIKey := s.cfg.LLM.APIKey
	if llmAPIKey == "" {
		llmAPIKey = s.cfg.OpenAI.APIKey
	}

	// Build upstream request body (strip Hystersis-only fields)
	upstreamPayload := map[string]interface{}{
		"model":    req.Model,
		"messages": req.Messages,
		"stream":   req.Stream,
	}
	if req.Temperature != nil {
		upstreamPayload["temperature"] = req.Temperature
	}
	if req.MaxTokens != nil {
		upstreamPayload["max_tokens"] = req.MaxTokens
	}
	if req.TopP != nil {
		upstreamPayload["top_p"] = req.TopP
	}

	reqBody, err := json.Marshal(upstreamPayload)
	if err != nil {
		http.Error(w, `{"error":{"message":"Failed to serialize request"}}`, http.StatusInternalServerError)
		return
	}

	upstreamURL := fmt.Sprintf("%s/v1/chat/completions", strings.TrimRight(llmBaseURL, "/"))
	upstreamReq, err := http.NewRequestWithContext(r.Context(), "POST", upstreamURL, bytes.NewReader(reqBody))
	if err != nil {
		http.Error(w, `{"error":{"message":"Failed to create upstream request"}}`, http.StatusInternalServerError)
		return
	}
	upstreamReq.Header.Set("Content-Type", "application/json")
	upstreamReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", llmAPIKey))

	// Streaming: pass headers and pipe response directly
	if req.Stream {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
	}

	resp, err := http.DefaultClient.Do(upstreamReq)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":{"message":"Upstream error: %s"}}`, err.Error()), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if req.Stream {
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body) //nolint:errcheck
		return
	}

	// Non-streaming: read, annotate with memory count, forward
	body, _ := io.ReadAll(resp.Body)

	var chatResp ChatCompletionResponse
	if json.Unmarshal(body, &chatResp) == nil && memoriesUsed > 0 {
		chatResp.MemoriesUsed = memoriesUsed
		augmented, _ := json.Marshal(chatResp)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		w.Write(augmented) //nolint:errcheck
		return
	}

	// Fallback: forward raw response unchanged
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(body) //nolint:errcheck
}
