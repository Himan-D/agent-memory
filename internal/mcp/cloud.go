package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

// CloudService provides the base URL and API key for the upstream memory API.
type CloudService interface {
	BaseURL() string
	APIKey() string
}

// CloudServer is the cloud-deployable MCP server with full tool parity,
// API key authentication, CORS support, and SSE streaming.
type CloudServer struct {
	upstream   CloudService
	httpServer *http.Server
	reqCount   atomic.Int64
}

// NewCloudServer creates a cloud MCP server listening on the given address.
func NewCloudServer(upstream CloudService, addr string) *CloudServer {
	cs := &CloudServer{upstream: upstream}

	mux := http.NewServeMux()

	// MCP endpoints
	mux.HandleFunc("/mcp", cs.handleMCP)
	mux.HandleFunc("/mcp/sse", cs.handleSSE)

	// Health
	mux.HandleFunc("/health", cs.handleHealth)
	mux.HandleFunc("/ready", cs.handleReady)

	// OAuth discovery (for MCP protocol compliance)
	mux.HandleFunc("/.well-known/oauth-protected-resource", cs.handleOAuthDiscovery)

	cs.httpServer = &http.Server{
		Addr:         addr,
		Handler:      cs.corsMiddleware(cs.authMiddleware(mux)),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second, // longer for SSE
		IdleTimeout:  120 * time.Second,
	}

	return cs
}

// Start begins listening for requests.
func (cs *CloudServer) Start() error {
	return cs.httpServer.ListenAndServe()
}

// Stop gracefully shuts down the server.
func (cs *CloudServer) Stop(ctx context.Context) error {
	return cs.httpServer.Shutdown(ctx)
}

// --- Middleware ---

func (cs *CloudServer) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key, X-Tenant-ID")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (cs *CloudServer) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for health endpoints and OPTIONS
		if r.URL.Path == "/health" || r.URL.Path == "/ready" || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		// Skip auth for OAuth discovery
		if r.URL.Path == "/.well-known/oauth-protected-resource" {
			next.ServeHTTP(w, r)
			return
		}

		expectedKey := os.Getenv("MCP_API_KEY")
		if expectedKey == "" {
			expectedKey = os.Getenv("HYSTERSIS_API_KEY")
		}

		if expectedKey == "" {
			writeJSONError(w, http.StatusInternalServerError, "server not configured: MCP_API_KEY or HYSTERSIS_API_KEY must be set")
			return
		}

		// Check X-API-Key header first, then Authorization: Bearer
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			auth := r.Header.Get("Authorization")
			if strings.HasPrefix(auth, "Bearer ") {
				apiKey = strings.TrimPrefix(auth, "Bearer ")
			}
		}

		if apiKey != expectedKey {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized: invalid or missing API key")
			return
		}

		// Inject tenant from X-Tenant-ID or the API key itself
		tenantID := r.Header.Get("X-Tenant-ID")
		if tenantID == "" {
			tenantID = apiKey
		}
		r = r.WithContext(context.WithValue(r.Context(), tenantIDKey{}, tenantID))

		next.ServeHTTP(w, r)
	})
}

// --- Tool Definitions ---

func (cs *CloudServer) allTools() []Tool {
	return []Tool{
		{
			Name:        "addMemory",
			Description: "Add a memory to the memory store. Use this to store important information.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"content":  map[string]string{"type": "string", "description": "The content to remember"},
					"userId":   map[string]string{"type": "string", "description": "User identifier"},
					"metadata": map[string]interface{}{"type": "object", "description": "Additional metadata"},
				},
				"required": []string{"content"},
			},
		},
		{
			Name:        "recall",
			Description: "Search memories by query. Find relevant memories for the current context.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query":  map[string]string{"type": "string", "description": "Search query"},
					"userId": map[string]string{"type": "string", "description": "Filter by user"},
					"limit":  map[string]interface{}{"type": "integer", "minimum": 1, "maximum": 50, "description": "Max results"},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "getMemories",
			Description: "Get all memories for a user.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"userId": map[string]string{"type": "string", "description": "User identifier"},
					"limit":  map[string]interface{}{"type": "integer", "description": "Max results"},
				},
			},
		},
		{
			Name:        "update_memory",
			Description: "Update an existing memory's content or metadata.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"memoryId": map[string]string{"type": "string", "description": "Memory ID to update"},
					"content":  map[string]string{"type": "string", "description": "New content"},
					"metadata": map[string]interface{}{"type": "object", "description": "Updated metadata"},
				},
				"required": []string{"memoryId"},
			},
		},
		{
			Name:        "deleteMemory",
			Description: "Delete a specific memory by ID.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"memoryId": map[string]string{"type": "string", "description": "Memory ID to delete"},
				},
				"required": []string{"memoryId"},
			},
		},
		{
			Name:        "delete_all_memories",
			Description: "Delete all memories for a user. Use with caution.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"userId": map[string]string{"type": "string", "description": "User whose memories to delete"},
				},
				"required": []string{"userId"},
			},
		},
		{
			Name:        "whoAmI",
			Description: "Get current user and authentication information.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "list_entities",
			Description: "List knowledge graph entities with optional type filtering.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"entityType": map[string]string{"type": "string", "description": "Filter by entity type"},
					"limit":      map[string]interface{}{"type": "integer", "description": "Max results"},
					"offset":     map[string]interface{}{"type": "integer", "description": "Pagination offset"},
				},
			},
		},
		{
			Name:        "delete_entities",
			Description: "Delete knowledge graph entities by ID or type.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"entityId":   map[string]string{"type": "string", "description": "Specific entity ID to delete"},
					"entityType": map[string]string{"type": "string", "description": "Delete all entities of this type"},
				},
			},
		},
		{
			Name:        "list_events",
			Description: "List recent system events and memory operations.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"limit":  map[string]interface{}{"type": "integer", "description": "Max events to return"},
					"offset": map[string]interface{}{"type": "integer", "description": "Pagination offset"},
					"type":   map[string]string{"type": "string", "description": "Filter by event type"},
				},
			},
		},
		{
			Name:        "get_event_status",
			Description: "Get the status of a specific async event or operation.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"eventId": map[string]string{"type": "string", "description": "Event ID to check"},
				},
				"required": []string{"eventId"},
			},
		},
	}
}

// --- Handlers ---

func (cs *CloudServer) handleMCP(w http.ResponseWriter, r *http.Request) {
	cs.reqCount.Add(1)

	if r.Method == http.MethodGet {
		// Return tool listing for GET requests
		cs.handleToolsList(w, r)
		return
	}

	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Parse JSON-RPC request
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONRPCError(w, nil, -32700, "parse error: could not read body")
		return
	}

	var req MCPRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		writeJSONRPCError(w, nil, -32700, "parse error: invalid JSON")
		return
	}

	switch req.Method {
	case "tools/list":
		cs.handleToolsList(w, r)
	case "tools/call":
		cs.handleToolCall(w, r, &req)
	case "initialize":
		cs.handleInitialize(w, r, &req)
	default:
		writeJSONRPCError(w, req.ID, -32601, fmt.Sprintf("method not found: %s", req.Method))
	}
}

func (cs *CloudServer) handleInitialize(w http.ResponseWriter, r *http.Request, req *MCPRequest) {
	result := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{
				"listChanged": false,
			},
		},
		"serverInfo": map[string]interface{}{
			"name":    "hystersis-cloud-mcp",
			"version": "1.0.0",
		},
	}

	writeJSONRPCResult(w, req.ID, result)
}

func (cs *CloudServer) handleToolsList(w http.ResponseWriter, r *http.Request) {
	tools := cs.allTools()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tools": tools,
	})
}

func (cs *CloudServer) handleToolCall(w http.ResponseWriter, r *http.Request, req *MCPRequest) {
	if req.Params == nil {
		writeJSONRPCError(w, req.ID, -32602, "invalid params: missing")
		return
	}

	var callParams struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}
	if err := json.Unmarshal(*req.Params, &callParams); err != nil {
		writeJSONRPCError(w, req.ID, -32602, "invalid params: "+err.Error())
		return
	}

	result, err := cs.dispatchTool(r.Context(), callParams.Name, callParams.Arguments)
	if err != nil {
		writeJSONRPCResult(w, req.ID, map[string]interface{}{
			"content": []map[string]interface{}{
				{"type": "text", "text": fmt.Sprintf("Error: %s", err.Error())},
			},
			"isError": true,
		})
		return
	}

	writeJSONRPCResult(w, req.ID, result)
}

func (cs *CloudServer) dispatchTool(ctx context.Context, name string, args map[string]interface{}) (interface{}, error) {
	baseURL := cs.upstream.BaseURL()

	switch name {
	case "addMemory":
		return cs.proxyPost(baseURL+"/memories", args)
	case "recall", "search":
		return cs.proxyPost(baseURL+"/search", args)
	case "getMemories":
		userID := stringParam(args, "userId", "default")
		return cs.proxyGet(fmt.Sprintf("%s/memories?user_id=%s&limit=%d", baseURL, userID, intParam(args, "limit", 10)))
	case "update_memory":
		memoryID := stringParam(args, "memoryId", "")
		if memoryID == "" {
			return nil, fmt.Errorf("memoryId is required")
		}
		payload := map[string]interface{}{}
		if c, ok := args["content"]; ok {
			payload["content"] = c
		}
		if m, ok := args["metadata"]; ok {
			payload["metadata"] = m
		}
		return cs.proxyPut(baseURL+"/memories/"+memoryID, payload)
	case "deleteMemory":
		memoryID := stringParam(args, "memoryId", "")
		if memoryID == "" {
			return nil, fmt.Errorf("memoryId is required")
		}
		return cs.proxyDelete(baseURL + "/memories/" + memoryID)
	case "delete_all_memories":
		userID := stringParam(args, "userId", "")
		if userID == "" {
			return nil, fmt.Errorf("userId is required")
		}
		return cs.proxyDelete(fmt.Sprintf("%s/memories?user_id=%s", baseURL, userID))
	case "whoAmI":
		return map[string]interface{}{
			"content": []map[string]interface{}{
				{"type": "text", "text": fmt.Sprintf("Tenant: %s\nStatus: active", tenantFromContext(ctx))},
			},
		}, nil
	case "list_entities":
		query := fmt.Sprintf("%s/entities?limit=%d&offset=%d", baseURL, intParam(args, "limit", 20), intParam(args, "offset", 0))
		if et, ok := args["entityType"].(string); ok && et != "" {
			query += "&entity_type=" + et
		}
		return cs.proxyGet(query)
	case "delete_entities":
		if eid, ok := args["entityId"].(string); ok && eid != "" {
			return cs.proxyDelete(baseURL + "/entities/" + eid)
		}
		if et, ok := args["entityType"].(string); ok && et != "" {
			return cs.proxyDelete(baseURL + "/entities?entity_type=" + et)
		}
		return nil, fmt.Errorf("entityId or entityType is required")
	case "list_events":
		query := fmt.Sprintf("%s/events?limit=%d&offset=%d", baseURL, intParam(args, "limit", 20), intParam(args, "offset", 0))
		if t, ok := args["type"].(string); ok && t != "" {
			query += "&type=" + t
		}
		return cs.proxyGet(query)
	case "get_event_status":
		eventID := stringParam(args, "eventId", "")
		if eventID == "" {
			return nil, fmt.Errorf("eventId is required")
		}
		return cs.proxyGet(baseURL + "/events/" + eventID)
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

// --- SSE Handler ---

func (cs *CloudServer) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Send initial connection event
	fmt.Fprintf(w, "event: endpoint\ndata: /mcp\n\n")
	flusher.Flush()

	// Keep connection alive with periodic heartbeats
	ctx := r.Context()
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fmt.Fprintf(w, ": heartbeat\n\n")
			flusher.Flush()
		}
	}
}

// --- Health ---

func (cs *CloudServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"service": "hystersis-cloud-mcp",
		"uptime":  cs.reqCount.Load(),
	})
}

func (cs *CloudServer) handleReady(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
}

func (cs *CloudServer) handleOAuthDiscovery(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Link", `</.well-known/oauth-protected-resource>; rel="protected-resource"`)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"oauth-protected-resource": map[string]interface{}{
			"authorization_endpoint": "/oauth/authorize",
			"token_endpoint":         "/oauth/token",
		},
	})
}

// --- Proxy Helpers ---

func (cs *CloudServer) proxyPost(url string, payload interface{}) (interface{}, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("cloud: marshal: %w", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("cloud: upstream: %w", err)
	}
	defer resp.Body.Close()

	return readUpstreamResponse(resp)
}

func (cs *CloudServer) proxyGet(url string) (interface{}, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("cloud: upstream: %w", err)
	}
	defer resp.Body.Close()

	return readUpstreamResponse(resp)
}

func (cs *CloudServer) proxyPut(url string, payload interface{}) (interface{}, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("cloud: marshal: %w", err)
	}

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("cloud: request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cloud: upstream: %w", err)
	}
	defer resp.Body.Close()

	return readUpstreamResponse(resp)
}

func (cs *CloudServer) proxyDelete(url string) (interface{}, error) {
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return nil, fmt.Errorf("cloud: request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cloud: upstream: %w", err)
	}
	defer resp.Body.Close()

	return readUpstreamResponse(resp)
}

func readUpstreamResponse(resp *http.Response) (interface{}, error) {
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("cloud: read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("upstream error (%d): %s", resp.StatusCode, string(data))
	}

	// Wrap the upstream response in MCP tool result format
	var result interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		// Return as raw text if not JSON
		return map[string]interface{}{
			"content": []map[string]interface{}{
				{"type": "text", "text": string(data)},
			},
		}, nil
	}

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{"type": "text", "text": string(data)},
		},
	}, nil
}

// --- JSON-RPC Helpers ---

func writeJSONRPCResult(w http.ResponseWriter, id interface{}, result interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(MCPResponse{
		JSONRPC: "2.0",
		Result:  result,
		ID:      id,
	})
}

func writeJSONRPCError(w http.ResponseWriter, id interface{}, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(MCPResponse{
		JSONRPC: "2.0",
		Error: &MCPCError{
			Code:    code,
			Message: message,
		},
		ID: id,
	})
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// --- Parameter Helpers ---

func stringParam(args map[string]interface{}, key, fallback string) string {
	if v, ok := args[key].(string); ok && v != "" {
		return v
	}
	return fallback
}

func intParam(args map[string]interface{}, key string, fallback int) int {
	if v, ok := args[key].(float64); ok {
		return int(v)
	}
	return fallback
}

func init() {
	// Ensure log output includes file/line for debugging cloud deployments
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}
