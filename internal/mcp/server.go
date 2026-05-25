package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"agent-memory/internal/memory"
	"agent-memory/internal/memory/types"
	"github.com/google/uuid"
)

type MCPServer struct {
	memSvc *memory.Service
	server *http.Server
}

type MCPRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method string          `json:"method"`
	Params *json.RawMessage `json:"params,omitempty"`
	ID     interface{}    `json:"id,omitempty"`
}

type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result interface{} `json:"result,omitempty"`
	Error *MCPCError  `json:"error,omitempty"`
	ID     interface{} `json:"id,omitempty"`
}

type MCPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type Tool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema any  `json:"inputSchema"`
}

type Resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description"`
	MimeType    string `json:"mimeType,omitempty"`
}

type ToolResult struct {
	Content []ContentBlock `json:"content"`
}

type ContentBlock struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	Resource string `json:"resource,omitempty"`
	Blob     string `json:"blob,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
}

func NewMCPServer(memSvc *memory.Service, port string) *MCPServer {
	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", handleMCP)
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/.well-known/oauth-protected-resource", handleOAuthDiscovery)

	server := &http.Server{
		Addr:         port,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return &MCPServer{
		memSvc:  memSvc,
		server: server,
	}
}

func handleMCP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var req MCPRequest
	if r.Method == http.MethodPost {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			json.NewEncoder(w).Encode(MCPResponse{
				JSONRPC: "2.0",
				Error: &MCPCError{
					Code:    -32700,
					Message: "Parse error",
				},
			})
			return
		}
	}

	switch r.URL.Query().Get("method") {
	case "tools/list":
		handleToolsList(w, r)
	case "resources/list":
		handleResourcesList(w, r)
	default:
		handleToolsList(w, r)
	}
}

func handleToolsList(w http.ResponseWriter, r *http.Request) {
	tools := []Tool{
		{
			Name:        "addMemory",
			Description: "Add a memory to the memory store. Use this to store important information you want to remember.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"content": map[string]string{"type": "string"},
					"userId": map[string]string{"type": "string"},
					"metadata": map[string]string{"type": "object"},
				},
				"required": []string{"content"},
			},
		},
		{
			Name:        "recall",
			Description: "Search memories by query. Use this to find relevant memories for the current context.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]string{"type": "string"},
					"limit": map[string]interface{}{"type": "integer", "minimum": 1, "maximum": 20},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "search",
			Description: "Search for memories. Alias for recall.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]string{"type": "string"},
					"limit": map[string]interface{}{"type": "integer"},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "whoAmI",
			Description: "Get current user information.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "getMemories",
			Description: "Get all memories for a user.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"userId": map[string]string{"type": "string"},
					"limit": map[string]interface{}{"type": "integer"},
				},
			},
		},
		{
			Name:        "deleteMemory",
			Description: "Delete a specific memory by ID.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"memoryId": map[string]string{"type": "string"},
				},
				"required": []string{"memoryId"},
			},
		},
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"tools": tools,
	})
}

func handleResourcesList(w http.ResponseWriter, r *http.Request) {
	resources := []Resource{
		{
			URI:         "profile://user",
			Name:        "User Profile",
			Description: "Get the user's profile information including preferences and recent activity",
			MimeType: "application/json",
		},
		{
			URI:         "memories://recent",
			Name:        "Recent Memories",
			Description: "Get recent memories",
			MimeType: "application/json",
		},
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"resources": resources,
	})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleOAuthDiscovery(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Link", `</.well-known/oauth-protected-resource>; rel="protected-resource"`)
	
	json.NewEncoder(w).Encode(map[string]interface{}{
		"oauth-protected-resource": map[string]interface{}{
			"authorization_endpoint": "/oauth/authorize",
			"token_endpoint": "/oauth/token",
		},
	})
}

func (s *MCPServer) Start() error {
	log.Printf("MCP server starting on %s", s.server.Addr)
	return s.server.ListenAndServe()
}

func (s *MCPServer) Stop(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

type ToolHandler struct {
	memSvc *memory.Service
}

func NewToolHandler(memSvc *memory.Service) *ToolHandler {
	return &ToolHandler{memSvc: memSvc}
}

func (h *ToolHandler) HandleToolCall(ctx context.Context, toolName string, params map[string]interface{}) (*ToolResult, error) {
	switch toolName {
	case "addMemory":
		return h.addMemory(ctx, params)
	case "recall", "search":
		return h.recall(ctx, params)
	case "whoAmI":
		return h.whoAmI(ctx, params)
	case "getMemories":
		return h.getMemories(ctx, params)
	case "deleteMemory":
		return h.deleteMemory(ctx, params)
	default:
		return nil, fmt.Errorf("unknown tool: %s", toolName)
	}
}

func (h *ToolHandler) addMemory(ctx context.Context, params map[string]interface{}) (*ToolResult, error) {
	content, ok := params["content"].(string)
	if !ok {
		return nil, fmt.Errorf("content is required")
	}

	userID := "default"
	if u, ok := params["userId"].(string); ok {
		userID = u
	}

	metadata := make(map[string]interface{})
	if m, ok := params["metadata"].(map[string]interface{}); ok {
		metadata = m
	}
	metadata["source"] = "mcp"

	mem := &types.Memory{
		ID:       uuid.New().String(),
		Content:  content,
		UserID:   userID,
		TenantID: "default",
		Type:     types.MemoryTypeUser,
		Metadata: metadata,
	}

	created, err := h.memSvc.CreateMemory(ctx, mem)
	if err != nil {
		return nil, fmt.Errorf("failed to create memory: %w", err)
	}

	return &ToolResult{
		Content: []ContentBlock{
			{
				Type: "text",
				Text: fmt.Sprintf("Memory added successfully with ID: %s", created.ID),
			},
		},
	}, nil
}

func (h *ToolHandler) recall(ctx context.Context, params map[string]interface{}) (*ToolResult, error) {
	query, ok := params["query"].(string)
	if !ok {
		return nil, fmt.Errorf("query is required")
	}

	limit := 5
	if l, ok := params["limit"].(float64); ok {
		limit = int(l)
	}

	userID := "default"
	if u, ok := params["userId"].(string); ok {
		userID = u
	}

	results, err := h.memSvc.SearchMemories(ctx, &types.SearchRequest{
		Query:  query,
		UserID: userID,
		Limit:  limit,
	})
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	var text strings.Builder
	for _, r := range results {
		text.WriteString(fmt.Sprintf("- %s (score: %.2f)\n", r.Text, r.Score))
	}

	if text.Len() == 0 {
		text.WriteString("No relevant memories found.")
	}

	return &ToolResult{
		Content: []ContentBlock{
			{
				Type: "text",
				Text: text.String(),
			},
		},
	}, nil
}

func (h *ToolHandler) whoAmI(ctx context.Context, params map[string]interface{}) (*ToolResult, error) {
	return &ToolResult{
		Content: []ContentBlock{
			{
				Type: "text",
				Text: "User: default\nRole: user\nStatus: active",
			},
		},
	}, nil
}

func (h *ToolHandler) getMemories(ctx context.Context, params map[string]interface{}) (*ToolResult, error) {
	limit := 10
	if l, ok := params["limit"].(float64); ok {
		limit = int(l)
	}

	userID := "default"
	if u, ok := params["userId"].(string); ok {
		userID = u
	}

	memories, err := h.memSvc.GetMemoriesByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get memories: %w", err)
	}

	if len(memories) > limit {
		memories = memories[:limit]
	}

	var text strings.Builder
	for _, m := range memories {
		text.WriteString(fmt.Sprintf("- %s\n", m.Content))
	}

	if text.Len() == 0 {
		text.WriteString("No memories found.")
	}

	return &ToolResult{
		Content: []ContentBlock{
			{
				Type: "text",
				Text: text.String(),
			},
		},
	}, nil
}

func (h *ToolHandler) deleteMemory(ctx context.Context, params map[string]interface{}) (*ToolResult, error) {
	memoryID, ok := params["memoryId"].(string)
	if !ok {
		return nil, fmt.Errorf("memoryId is required")
	}

	err := h.memSvc.DeleteMemory(ctx, memoryID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete memory: %w", err)
	}

	return &ToolResult{
		Content: []ContentBlock{
			{
				Type: "text",
				Text: fmt.Sprintf("Memory %s deleted successfully", memoryID),
			},
		},
	}, nil
}