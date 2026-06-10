package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
)

var (
	port         = flag.String("port", "8082", "Port to listen on")
	memoryAPIURL = flag.String("memory-api", "http://localhost:8081", "Memory API URL")
	enableOAuth  = flag.Bool("oauth", false, "Enable OAuth authentication")
	oauthSecret  = flag.String("oauth-secret", "default-secret", "OAuth secret key")
)

// ==================== JSON-RPC 2.0 Types ====================

type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type MCPTool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"inputSchema"`
}

// ==================== Response Recorder ====================

type responseRecorder struct {
	headers http.Header
	body    bytes.Buffer
	status  int
}

func newResponseRecorder() *responseRecorder {
	return &responseRecorder{headers: make(http.Header), status: http.StatusOK}
}

func (r *responseRecorder) Header() http.Header         { return r.headers }
func (r *responseRecorder) Write(b []byte) (int, error) { return r.body.Write(b) }
func (r *responseRecorder) WriteHeader(code int)        { r.status = code }

// ==================== Server ====================

type MCPServer struct {
	memoryAPIURL string
	httpServer   *http.Server
	mu           sync.RWMutex
	sseClients   map[string]chan []byte
}

func NewMCPServer() *MCPServer {
	s := &MCPServer{
		memoryAPIURL: *memoryAPIURL,
		sseClients:   make(map[string]chan []byte),
	}

	mux := http.NewServeMux()

	// MCP Protocol (JSON-RPC 2.0 + SSE transport)
	mux.HandleFunc("/mcp", s.handleMCP)
	mux.HandleFunc("/sse", s.handleSSE)
	mux.HandleFunc("/message", s.handleMessage)

	// OAuth
	mux.HandleFunc("/oauth/authorize", handleOAuthAuthorize)
	mux.HandleFunc("/oauth/token", handleOAuthToken)
	mux.HandleFunc("/.well-known/oauth-protected-resource", handleOAuthDiscovery)

	// Tools (via HTTP to memory-api)
	mux.HandleFunc("/tools/addMemory", handleAddMemory)
	mux.HandleFunc("/tools/recall", handleRecall)
	mux.HandleFunc("/tools/search", handleSearch)
	mux.HandleFunc("/tools/whoAmI", handleWhoAmI)
	mux.HandleFunc("/tools/getMemories", handleGetMemories)
	mux.HandleFunc("/tools/deleteMemory", handleDeleteMemory)
	mux.HandleFunc("/tools/updateMemory", handleUpdateMemory)
	mux.HandleFunc("/tools/getMemory", handleGetMemory)
	mux.HandleFunc("/tools/listEntities", handleListEntities)
	mux.HandleFunc("/tools/createEntity", handleCreateEntity)
	mux.HandleFunc("/tools/createRelation", handleCreateRelation)
	mux.HandleFunc("/tools/getEntityRelations", handleGetEntityRelations)
	mux.HandleFunc("/tools/addFeedback", handleAddFeedback)
	mux.HandleFunc("/tools/getMemoryHistory", handleGetMemoryHistory)
	mux.HandleFunc("/tools/createSession", handleCreateSession)
	mux.HandleFunc("/tools/getContext", handleGetContext)
	mux.HandleFunc("/tools/createSkill", handleCreateSkill)
	mux.HandleFunc("/tools/listSkills", handleListSkills)
	mux.HandleFunc("/tools/temporalSearch", handleTemporalSearch)
	mux.HandleFunc("/tools/getCompressionStats", handleGetCompressionStats)
	mux.HandleFunc("/tools/setTierPolicy", handleSetTierPolicy)
	mux.HandleFunc("/tools/getProvenance", handleGetProvenance)
	mux.HandleFunc("/tools/createConcept", handleCreateConcept)
	mux.HandleFunc("/tools/linkToConcept", handleLinkToConcept)
	mux.HandleFunc("/tools/setReminder", handleSetReminder)
	mux.HandleFunc("/tools/checkSafety", handleCheckSafety)

	// Health
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/ready", handleReady)

	s.httpServer = &http.Server{
		Addr:         ":" + *port,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return s
}

func (s *MCPServer) Start() error {
	log.Printf("MCP Server starting on %s", *port)
	log.Printf("Memory API endpoint: %s", s.memoryAPIURL)

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("MCP server: %w", err)
	}
	return nil
}

func (s *MCPServer) Stop(ctx context.Context) error {
	log.Println("Shutting down MCP Server...")
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	return nil
}

func main() {
	flag.Parse()

	server := NewMCPServer()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received shutdown signal")
		server.Stop(ctx)
	}()

	if err := server.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// ==================== MCP Protocol ====================

func (s *MCPServer) getToolList() []MCPTool {
	return []MCPTool{
		{
			Name:        "add_memory",
			Description: "Store a new memory with optional metadata",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"content":    map[string]interface{}{"type": "string", "description": "Memory content to store"},
					"user_id":    map[string]interface{}{"type": "string", "description": "User ID scope"},
					"agent_id":   map[string]interface{}{"type": "string", "description": "Agent ID scope"},
					"session_id": map[string]interface{}{"type": "string", "description": "Session ID scope"},
					"metadata":   map[string]interface{}{"type": "object", "description": "Additional metadata"},
				},
				"required": []string{"content"},
			},
		},
		{
			Name:        "recall",
			Description: "Search memories using semantic similarity",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{"type": "string", "description": "Search query"},
					"limit": map[string]interface{}{"type": "integer", "description": "Max results to return", "default": 5},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "search",
			Description: "Search memories (alias for recall)",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{"type": "string", "description": "Search query"},
					"limit": map[string]interface{}{"type": "integer", "description": "Max results to return", "default": 5},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "get_memory",
			Description: "Retrieve a specific memory by ID",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"memoryId": map[string]interface{}{"type": "string", "description": "Memory ID to retrieve"},
				},
				"required": []string{"memoryId"},
			},
		},
		{
			Name:        "update_memory",
			Description: "Update an existing memory's content or metadata",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"memoryId": map[string]interface{}{"type": "string", "description": "Memory ID to update"},
					"content":  map[string]interface{}{"type": "string", "description": "New content"},
					"metadata": map[string]interface{}{"type": "object", "description": "Updated metadata"},
				},
				"required": []string{"memoryId", "content"},
			},
		},
		{
			Name:        "delete_memory",
			Description: "Delete a memory by ID",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"memoryId": map[string]interface{}{"type": "string", "description": "Memory ID to delete"},
				},
				"required": []string{"memoryId"},
			},
		},
		{
			Name:        "get_memories",
			Description: "List all memories",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "who_am_i",
			Description: "Get current authenticated user info",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "list_entities",
			Description: "List knowledge graph entities",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"entityType": map[string]interface{}{"type": "string", "description": "Filter by entity type"},
					"limit":      map[string]interface{}{"type": "integer", "description": "Max results"},
					"offset":     map[string]interface{}{"type": "integer", "description": "Pagination offset"},
				},
			},
		},
		{
			Name:        "add_entity",
			Description: "Create a knowledge graph entity",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name":       map[string]interface{}{"type": "string", "description": "Entity name"},
					"type":       map[string]interface{}{"type": "string", "description": "Entity type"},
					"properties": map[string]interface{}{"type": "object", "description": "Entity properties"},
				},
				"required": []string{"name", "type"},
			},
		},
		{
			Name:        "create_relation",
			Description: "Create a relation between two entities",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"from_id":  map[string]interface{}{"type": "string", "description": "Source entity ID"},
					"to_id":    map[string]interface{}{"type": "string", "description": "Target entity ID"},
					"type":     map[string]interface{}{"type": "string", "description": "Relation type"},
					"metadata": map[string]interface{}{"type": "object", "description": "Relation metadata"},
				},
				"required": []string{"from_id", "to_id", "type"},
			},
		},
		{
			Name:        "get_entity_relations",
			Description: "Get all relations for an entity",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"entityId": map[string]interface{}{"type": "string", "description": "Entity ID"},
					"type":     map[string]interface{}{"type": "string", "description": "Filter by relation type"},
				},
				"required": []string{"entityId"},
			},
		},
		{
			Name:        "add_feedback",
			Description: "Add feedback to a memory",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"memoryId": map[string]interface{}{"type": "string", "description": "Memory ID"},
					"type":     map[string]interface{}{"type": "string", "description": "Feedback type (positive/negative)"},
					"comment":  map[string]interface{}{"type": "string", "description": "Optional comment"},
					"userId":   map[string]interface{}{"type": "string", "description": "User providing feedback"},
				},
				"required": []string{"memoryId", "type"},
			},
		},
		{
			Name:        "get_memory_history",
			Description: "Get modification history for a memory",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"memoryId": map[string]interface{}{"type": "string", "description": "Memory ID"},
				},
				"required": []string{"memoryId"},
			},
		},
		{
			Name:        "create_session",
			Description: "Create a new conversation session",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"agentId":  map[string]interface{}{"type": "string", "description": "Agent ID for this session"},
					"metadata": map[string]interface{}{"type": "object", "description": "Session metadata"},
				},
				"required": []string{"agentId"},
			},
		},
		{
			Name:        "get_context",
			Description: "Get conversation context for a session",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"sessionId": map[string]interface{}{"type": "string", "description": "Session ID"},
					"limit":     map[string]interface{}{"type": "integer", "description": "Max messages to return", "default": 50},
				},
				"required": []string{"sessionId"},
			},
		},
		{
			Name:        "create_skill",
			Description: "Create a new skill",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name":        map[string]interface{}{"type": "string", "description": "Skill name"},
					"description": map[string]interface{}{"type": "string", "description": "Skill description"},
					"domain":      map[string]interface{}{"type": "string", "description": "Skill domain"},
				},
				"required": []string{"name"},
			},
		},
		{
			Name:        "list_skills",
			Description: "List available skills",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"domain": map[string]interface{}{"type": "string", "description": "Filter by domain"},
					"limit":  map[string]interface{}{"type": "integer", "description": "Max results"},
					"offset": map[string]interface{}{"type": "integer", "description": "Pagination offset"},
				},
			},
		},
		{
			Name:        "temporal_search",
			Description: "Search memories within a time range",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query":      map[string]interface{}{"type": "string", "description": "Search query"},
					"time_start": map[string]interface{}{"type": "string", "description": "Start time (ISO 8601)"},
					"time_end":   map[string]interface{}{"type": "string", "description": "End time (ISO 8601)"},
					"limit":      map[string]interface{}{"type": "integer", "description": "Max results", "default": 10},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "get_compression_stats",
			Description: "Get compression engine statistics",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "set_tier_policy",
			Description: "Set the memory tier policy (balanced/aggressive/conservative)",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"policy": map[string]interface{}{"type": "string", "description": "Tier policy: balanced, aggressive, or conservative"},
				},
				"required": []string{"policy"},
			},
		},
		{
			Name:        "get_provenance",
			Description: "Get provenance chain for a memory",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"memoryId": map[string]interface{}{"type": "string", "description": "Memory ID"},
				},
				"required": []string{"memoryId"},
			},
		},
		{
			Name:        "create_concept",
			Description: "Create a concept node in the knowledge graph",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name":        map[string]interface{}{"type": "string", "description": "Concept name"},
					"description": map[string]interface{}{"type": "string", "description": "Concept description"},
					"properties":  map[string]interface{}{"type": "object", "description": "Concept properties"},
				},
				"required": []string{"name"},
			},
		},
		{
			Name:        "link_to_concept",
			Description: "Link a memory or entity to a concept",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"concept_id": map[string]interface{}{"type": "string", "description": "Concept ID"},
					"node_id":    map[string]interface{}{"type": "string", "description": "Memory or entity ID to link"},
					"rel_type":   map[string]interface{}{"type": "string", "description": "Relation type"},
				},
				"required": []string{"concept_id", "node_id"},
			},
		},
		{
			Name:        "set_reminder",
			Description: "Set a prospective memory reminder on a memory",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"memory_id": map[string]interface{}{"type": "string", "description": "Memory ID"},
					"remind_at": map[string]interface{}{"type": "string", "description": "Reminder time (ISO 8601)"},
					"condition": map[string]interface{}{"type": "string", "description": "Optional trigger condition"},
				},
				"required": []string{"memory_id", "remind_at"},
			},
		},
		{
			Name:        "check_safety",
			Description: "Check content safety classification",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"content": map[string]interface{}{"type": "string", "description": "Content to check"},
				},
				"required": []string{"content"},
			},
		},
	}
}

// processRPCRequest handles a JSON-RPC 2.0 request and returns the serialized response.
func (s *MCPServer) processRPCRequest(req JSONRPCRequest) []byte {
	var (
		result interface{}
		rpcErr *RPCError
	)

	switch req.Method {
	case "initialize":
		result = map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			"serverInfo": map[string]interface{}{
				"name":    "hystersis-memory",
				"version": "1.0.0",
			},
		}
	case "notifications/initialized":
		result = map[string]interface{}{}
	case "tools/list":
		result = map[string]interface{}{"tools": s.getToolList()}
	case "tools/call":
		var p struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			rpcErr = &RPCError{Code: -32602, Message: "invalid params: " + err.Error()}
		} else {
			res, err := s.callTool(p.Name, p.Arguments)
			if err != nil {
				rpcErr = &RPCError{Code: -32603, Message: err.Error()}
			} else {
				result = map[string]interface{}{
					"content": []map[string]interface{}{
						{"type": "text", "text": toJSON(res)},
					},
				}
			}
		}
	default:
		// Legacy backward compat: route by method name directly
		res, err := s.legacyDispatch(req)
		if err != nil {
			rpcErr = &RPCError{Code: -32601, Message: "method not found: " + req.Method}
		} else {
			result = res
		}
	}

	resp := JSONRPCResponse{JSONRPC: "2.0", ID: req.ID}
	if rpcErr != nil {
		resp.Error = rpcErr
	} else {
		resp.Result = result
	}

	b, _ := json.Marshal(resp)
	return b
}

func (s *MCPServer) handleMCP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"tools": s.getToolList()})
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req JSONRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONRPCError(w, nil, -32700, "Parse error: "+err.Error())
		return
	}
	if req.JSONRPC == "" {
		req.JSONRPC = "2.0"
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(s.processRPCRequest(req))
}

func (s *MCPServer) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	sessionID := uuid.New().String()
	msgCh := make(chan []byte, 16)

	s.mu.Lock()
	s.sseClients[sessionID] = msgCh
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.sseClients, sessionID)
		s.mu.Unlock()
	}()

	fmt.Fprintf(w, "event: endpoint\ndata: /message?sessionId=%s\n\n", sessionID)
	flusher.Flush()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-msgCh:
			fmt.Fprintf(w, "event: message\ndata: %s\n\n", msg)
			flusher.Flush()
		}
	}
}

func (s *MCPServer) handleMessage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST,OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.URL.Query().Get("sessionId")

	var req JSONRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONRPCError(w, nil, -32700, "Parse error: "+err.Error())
		return
	}
	if req.JSONRPC == "" {
		req.JSONRPC = "2.0"
	}

	respBytes := s.processRPCRequest(req)

	if sessionID != "" {
		s.mu.RLock()
		ch, ok := s.sseClients[sessionID]
		s.mu.RUnlock()
		if ok {
			select {
			case ch <- respBytes:
			default:
				// channel full — client is not draining; skip
			}
			w.WriteHeader(http.StatusAccepted)
			return
		}
	}

	// No SSE session: respond directly
	w.Header().Set("Content-Type", "application/json")
	w.Write(respBytes)
}

// callTool invokes a named tool by dispatching to its existing handler via a
// synthetic HTTP request, capturing the response with responseRecorder.
func (s *MCPServer) callTool(name string, args json.RawMessage) (interface{}, error) {
	canonical := normalizeTool(name)

	if args == nil {
		args = json.RawMessage("{}")
	}

	fakeReq, _ := http.NewRequest("POST", "/", bytes.NewReader(args))
	fakeReq.Header.Set("Content-Type", "application/json")

	rec := newResponseRecorder()

	switch canonical {
	case "addMemory":
		handleAddMemory(rec, fakeReq)
	case "recall", "search":
		handleRecall(rec, fakeReq)
	case "whoAmI":
		handleWhoAmI(rec, fakeReq)
	case "getMemories":
		handleGetMemories(rec, fakeReq)
	case "deleteMemory":
		handleDeleteMemory(rec, fakeReq)
	case "updateMemory":
		handleUpdateMemory(rec, fakeReq)
	case "getMemory":
		handleGetMemory(rec, fakeReq)
	case "listEntities":
		handleListEntities(rec, fakeReq)
	case "createEntity":
		handleCreateEntity(rec, fakeReq)
	case "createRelation":
		handleCreateRelation(rec, fakeReq)
	case "getEntityRelations":
		handleGetEntityRelations(rec, fakeReq)
	case "addFeedback":
		handleAddFeedback(rec, fakeReq)
	case "getMemoryHistory":
		handleGetMemoryHistory(rec, fakeReq)
	case "createSession":
		handleCreateSession(rec, fakeReq)
	case "getContext":
		handleGetContext(rec, fakeReq)
	case "createSkill":
		handleCreateSkill(rec, fakeReq)
	case "listSkills":
		handleListSkills(rec, fakeReq)
	case "temporalSearch":
		handleTemporalSearch(rec, fakeReq)
	case "getCompressionStats":
		handleGetCompressionStats(rec, fakeReq)
	case "setTierPolicy":
		handleSetTierPolicy(rec, fakeReq)
	case "getProvenance":
		handleGetProvenance(rec, fakeReq)
	case "createConcept":
		handleCreateConcept(rec, fakeReq)
	case "linkToConcept":
		handleLinkToConcept(rec, fakeReq)
	case "setReminder":
		handleSetReminder(rec, fakeReq)
	case "checkSafety":
		handleCheckSafety(rec, fakeReq)
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}

	if rec.status >= 400 {
		return nil, fmt.Errorf("tool error (HTTP %d): %s", rec.status, strings.TrimSpace(rec.body.String()))
	}

	var result interface{}
	if err := json.Unmarshal(rec.body.Bytes(), &result); err != nil {
		return rec.body.String(), nil
	}
	return result, nil
}

// legacyDispatch handles legacy {"method":"addMemory","params":{}} requests.
func (s *MCPServer) legacyDispatch(req JSONRPCRequest) (interface{}, error) {
	args := req.Params
	if args == nil {
		args = json.RawMessage("{}")
	}
	return s.callTool(req.Method, args)
}

// normalizeTool maps snake_case MCP tool names to the camelCase internal names.
// camelCase names pass through unchanged.
func normalizeTool(name string) string {
	switch name {
	case "add_memory":
		return "addMemory"
	case "who_am_i":
		return "whoAmI"
	case "get_memories":
		return "getMemories"
	case "delete_memory":
		return "deleteMemory"
	case "update_memory":
		return "updateMemory"
	case "get_memory":
		return "getMemory"
	case "list_entities":
		return "listEntities"
	case "add_entity":
		return "createEntity"
	case "create_relation":
		return "createRelation"
	case "get_entity_relations":
		return "getEntityRelations"
	case "add_feedback":
		return "addFeedback"
	case "get_memory_history":
		return "getMemoryHistory"
	case "create_session":
		return "createSession"
	case "get_context":
		return "getContext"
	case "create_skill":
		return "createSkill"
	case "list_skills":
		return "listSkills"
	case "temporal_search":
		return "temporalSearch"
	case "get_compression_stats":
		return "getCompressionStats"
	case "set_tier_policy":
		return "setTierPolicy"
	case "get_provenance":
		return "getProvenance"
	case "create_concept":
		return "createConcept"
	case "link_to_concept":
		return "linkToConcept"
	case "set_reminder":
		return "setReminder"
	case "check_safety":
		return "checkSafety"
	default:
		return name // camelCase or unknown — pass through
	}
}

// ==================== JSON-RPC Helpers ====================

func writeJSONRPCError(w http.ResponseWriter, id interface{}, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &RPCError{Code: code, Message: msg},
	})
}

func toJSON(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}

// ==================== Handlers ====================

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "ok", "service": "mcp-server"}`))
}

func handleReady(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "ready"}`))
}

// Tool Handlers - delegate to Memory API

func handleAddMemory(w http.ResponseWriter, r *http.Request) {
	var params struct {
		Content string `json:"content"`
		UserID  string `json:"userId,omitempty"`
	}
	json.NewDecoder(r.Body).Decode(&params)

	// Call Memory API
	resp, err := callMemoryAPI("/memories", map[string]interface{}{
		"content": params.Content,
		"user_id": params.UserID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
}

func handleRecall(w http.ResponseWriter, r *http.Request) {
	var params struct {
		Query string `json:"query"`
		Limit int    `json:"limit,omitempty"`
	}
	json.NewDecoder(r.Body).Decode(&params)

	limit := 5
	if params.Limit > 0 {
		limit = params.Limit
	}

	// Call Memory API
	resp, err := callMemoryAPI("/search", map[string]interface{}{
		"query": params.Query,
		"limit": limit,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
}

func handleSearch(w http.ResponseWriter, r *http.Request) {
	handleRecall(w, r)
}

func handleWhoAmI(w http.ResponseWriter, r *http.Request) {
	auth := r.Header.Get("Authorization")
	apiKey := ""
	if strings.HasPrefix(auth, "Bearer ") {
		apiKey = strings.TrimPrefix(auth, "Bearer ")
	} else if strings.HasPrefix(auth, "bearer ") {
		apiKey = strings.TrimPrefix(auth, "bearer ")
	}

	prefix := ""
	if len(apiKey) >= 8 {
		prefix = apiKey[:8]
	} else if len(apiKey) > 0 {
		prefix = apiKey
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"userId":         "authenticated",
		"role":           "user",
		"status":         "active",
		"api_key_prefix": prefix,
	})
}

func handleGetMemories(w http.ResponseWriter, r *http.Request) {
	resp, err := callMemoryAPI("/memories", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
}

func handleDeleteMemory(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var params struct {
		MemoryID string `json:"memoryId"`
	}
	json.Unmarshal(bodyBytes, &params)

	// Call Memory API with DELETE method
	url := *memoryAPIURL + "/memories/" + params.MemoryID
	req, _ := http.NewRequest("DELETE", url, nil)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func handleUpdateMemory(w http.ResponseWriter, r *http.Request) {
	var params struct {
		MemoryID string                 `json:"memoryId"`
		Content  string                 `json:"content"`
		Metadata map[string]interface{} `json:"metadata,omitempty"`
	}
	json.NewDecoder(r.Body).Decode(&params)

	payload := map[string]interface{}{
		"content": params.Content,
	}
	if params.Metadata != nil {
		payload["metadata"] = params.Metadata
	}

	url := *memoryAPIURL + "/memories/" + params.MemoryID
	b, _ := json.Marshal(payload)
	req, _ := http.NewRequest("PUT", url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func handleGetMemory(w http.ResponseWriter, r *http.Request) {
	var params struct {
		MemoryID string `json:"memoryId"`
	}
	json.NewDecoder(r.Body).Decode(&params)

	url := *memoryAPIURL + "/memories/" + params.MemoryID
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func handleListEntities(w http.ResponseWriter, r *http.Request) {
	var params struct {
		EntityType string `json:"entityType,omitempty"`
		Limit      int    `json:"limit,omitempty"`
		Offset     int    `json:"offset,omitempty"`
	}
	json.NewDecoder(r.Body).Decode(&params)

	url := *memoryAPIURL + "/entities"
	query := fmt.Sprintf("?limit=%d&offset=%d", params.Limit, params.Offset)
	if params.EntityType != "" {
		query += "&entity_type=" + params.EntityType
	}
	req, _ := http.NewRequest("GET", url+query, nil)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func handleCreateEntity(w http.ResponseWriter, r *http.Request) {
	var params struct {
		Name       string                 `json:"name"`
		Type       string                 `json:"type"`
		Properties map[string]interface{} `json:"properties,omitempty"`
	}
	json.NewDecoder(r.Body).Decode(&params)

	payload := map[string]interface{}{
		"name": params.Name,
		"type": params.Type,
	}
	if params.Properties != nil {
		payload["properties"] = params.Properties
	}

	resp, err := callMemoryAPI("/entities", payload)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
}

func handleCreateRelation(w http.ResponseWriter, r *http.Request) {
	var params struct {
		FromID   string                 `json:"from_id"`
		ToID     string                 `json:"to_id"`
		Type     string                 `json:"type"`
		Metadata map[string]interface{} `json:"metadata,omitempty"`
	}
	json.NewDecoder(r.Body).Decode(&params)

	payload := map[string]interface{}{
		"from_id": params.FromID,
		"to_id":   params.ToID,
		"type":    params.Type,
	}
	if params.Metadata != nil {
		payload["metadata"] = params.Metadata
	}

	resp, err := callMemoryAPI("/relations", payload)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
}

func handleGetEntityRelations(w http.ResponseWriter, r *http.Request) {
	var params struct {
		EntityID string `json:"entityId"`
		Type     string `json:"type,omitempty"`
	}
	json.NewDecoder(r.Body).Decode(&params)

	url := *memoryAPIURL + "/entities/" + params.EntityID + "/relations"
	if params.Type != "" {
		url += "?type=" + params.Type
	}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func handleAddFeedback(w http.ResponseWriter, r *http.Request) {
	var params struct {
		MemoryID string `json:"memoryId"`
		Type     string `json:"type"`
		Comment  string `json:"comment,omitempty"`
		UserID   string `json:"userId,omitempty"`
	}
	json.NewDecoder(r.Body).Decode(&params)

	payload := map[string]interface{}{
		"memory_id": params.MemoryID,
		"type":      params.Type,
	}
	if params.Comment != "" {
		payload["comment"] = params.Comment
	}
	if params.UserID != "" {
		payload["user_id"] = params.UserID
	}

	resp, err := callMemoryAPI("/feedback", payload)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
}

func handleGetMemoryHistory(w http.ResponseWriter, r *http.Request) {
	var params struct {
		MemoryID string `json:"memoryId"`
	}
	json.NewDecoder(r.Body).Decode(&params)

	url := *memoryAPIURL + "/memories/" + params.MemoryID + "/history"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func handleCreateSession(w http.ResponseWriter, r *http.Request) {
	var params struct {
		AgentID  string                 `json:"agentId"`
		Metadata map[string]interface{} `json:"metadata,omitempty"`
	}
	json.NewDecoder(r.Body).Decode(&params)

	payload := map[string]interface{}{
		"agent_id": params.AgentID,
	}
	if params.Metadata != nil {
		payload["metadata"] = params.Metadata
	}

	resp, err := callMemoryAPI("/sessions", payload)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
}

func handleGetContext(w http.ResponseWriter, r *http.Request) {
	var params struct {
		SessionID string `json:"sessionId"`
		Limit     int    `json:"limit,omitempty"`
	}
	json.NewDecoder(r.Body).Decode(&params)

	limit := 50
	if params.Limit > 0 {
		limit = params.Limit
	}

	url := fmt.Sprintf("%s/sessions/%s/messages?limit=%d", *memoryAPIURL, params.SessionID, limit)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func handleCreateSkill(w http.ResponseWriter, r *http.Request) {
	var params map[string]interface{}
	json.NewDecoder(r.Body).Decode(&params)

	resp, err := callMemoryAPI("/skills", params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
}

func handleListSkills(w http.ResponseWriter, r *http.Request) {
	var params struct {
		Domain string `json:"domain,omitempty"`
		Limit  int    `json:"limit,omitempty"`
		Offset int    `json:"offset,omitempty"`
	}
	json.NewDecoder(r.Body).Decode(&params)

	url := *memoryAPIURL + "/skills"
	query := fmt.Sprintf("?limit=%d&offset=%d", params.Limit, params.Offset)
	if params.Domain != "" {
		query += "&domain=" + params.Domain
	}
	req, _ := http.NewRequest("GET", url+query, nil)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func handleTemporalSearch(w http.ResponseWriter, r *http.Request) {
	var params struct {
		Query     string `json:"query"`
		TimeStart string `json:"time_start,omitempty"`
		TimeEnd   string `json:"time_end,omitempty"`
		Limit     int    `json:"limit,omitempty"`
	}
	json.NewDecoder(r.Body).Decode(&params)

	limit := 10
	if params.Limit > 0 {
		limit = params.Limit
	}

	url := fmt.Sprintf("%s/search?q=%s&limit=%d", *memoryAPIURL, params.Query, limit)
	if params.TimeStart != "" {
		url += "&time_start=" + params.TimeStart
	}
	if params.TimeEnd != "" {
		url += "&time_end=" + params.TimeEnd
	}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func handleGetCompressionStats(w http.ResponseWriter, r *http.Request) {
	url := *memoryAPIURL + "/compression/stats"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func handleSetTierPolicy(w http.ResponseWriter, r *http.Request) {
	var params struct {
		Policy string `json:"policy"`
	}
	json.NewDecoder(r.Body).Decode(&params)

	url := *memoryAPIURL + "/tier/policy"
	b, _ := json.Marshal(map[string]interface{}{"policy": params.Policy})
	req, _ := http.NewRequest("PUT", url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func handleGetProvenance(w http.ResponseWriter, r *http.Request) {
	var params struct {
		MemoryID string `json:"memoryId"`
	}
	json.NewDecoder(r.Body).Decode(&params)

	url := *memoryAPIURL + "/memories/" + params.MemoryID + "/versions"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func handleCreateConcept(w http.ResponseWriter, r *http.Request) {
	var params map[string]interface{}
	json.NewDecoder(r.Body).Decode(&params)

	resp, err := callMemoryAPI("/concepts", params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
}

func handleLinkToConcept(w http.ResponseWriter, r *http.Request) {
	var params struct {
		ConceptID string `json:"concept_id"`
		NodeID    string `json:"node_id"`
		RelType   string `json:"rel_type,omitempty"`
	}
	json.NewDecoder(r.Body).Decode(&params)

	url := *memoryAPIURL + "/concepts/" + params.ConceptID + "/link"
	payload := map[string]interface{}{
		"node_id":  params.NodeID,
		"rel_type": params.RelType,
	}
	b, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func handleSetReminder(w http.ResponseWriter, r *http.Request) {
	var params struct {
		MemoryID  string `json:"memory_id"`
		RemindAt  string `json:"remind_at"`
		Condition string `json:"condition,omitempty"`
	}
	json.NewDecoder(r.Body).Decode(&params)

	url := *memoryAPIURL + "/memories/" + params.MemoryID + "/remind"
	payload := map[string]interface{}{
		"remind_at": params.RemindAt,
		"condition": params.Condition,
	}
	b, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func handleCheckSafety(w http.ResponseWriter, r *http.Request) {
	var params map[string]interface{}
	json.NewDecoder(r.Body).Decode(&params)

	resp, err := callMemoryAPI("/safety/check", params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
}

// ==================== OAuth Handlers ====================

func handleOAuthDiscovery(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Link", `</.well-known/oauth-protected-resource>; rel="protected-resource"`)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"oauth-protected-resource": map[string]interface{}{
			"authorization_endpoint": "/oauth/authorize",
			"token_endpoint":         "/oauth/token",
		},
	})
}

// validAPIKey returns the configured API key from the environment.
// It checks MCP_API_KEY first, then falls back to HYSTERSIS_API_KEY.
func validAPIKey() string {
	if key := os.Getenv("MCP_API_KEY"); key != "" {
		return key
	}
	return os.Getenv("HYSTERSIS_API_KEY")
}

// requireAPIKey validates the Authorization: Bearer <key> header against the
// configured API key. Returns false and writes a 401 response if invalid.
func requireAPIKey(w http.ResponseWriter, r *http.Request) bool {
	expected := validAPIKey()
	if expected == "" {
		// No key configured — deny all requests to prevent accidental open access
		http.Error(w, "server not configured: MCP_API_KEY or HYSTERSIS_API_KEY must be set", http.StatusUnauthorized)
		return false
	}

	auth := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(auth, prefix) {
		http.Error(w, "unauthorized: missing or malformed Authorization header", http.StatusUnauthorized)
		return false
	}

	provided := strings.TrimPrefix(auth, prefix)
	if provided != expected {
		http.Error(w, "unauthorized: invalid API key", http.StatusUnauthorized)
		return false
	}

	return true
}

func handleOAuthAuthorize(w http.ResponseWriter, r *http.Request) {
	if !requireAPIKey(w, r) {
		return
	}

	state := r.URL.Query().Get("state")
	redirectURI := r.URL.Query().Get("redirect_uri")

	if redirectURI != "" {
		http.Redirect(w, r, redirectURI+"?code="+validAPIKey()+"&state="+state, http.StatusFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"auth_url": "/oauth/authorize",
		"state":    state,
	})
}

func handleOAuthToken(w http.ResponseWriter, r *http.Request) {
	if !requireAPIKey(w, r) {
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"access_token":  validAPIKey(),
		"token_type":    "Bearer",
		"expires_in":    3600,
		"refresh_token": validAPIKey(),
	})
}

// ==================== Helpers ====================

func callMemoryAPI(path string, payload interface{}) ([]byte, error) {
	url := *memoryAPIURL + path

	var body io.Reader
	if payload != nil {
		b, _ := json.Marshal(payload)
		body = bytes.NewReader(b)
	}

	req, _ := http.NewRequest("POST", url, body)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// routeByMethod dispatches a named method to its handler (legacy HTTP path).
func routeByMethod(w http.ResponseWriter, r *http.Request, method string, bodyBytes []byte) {
	r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	switch method {
	case "addMemory":
		handleAddMemory(w, r)
	case "recall", "search":
		handleRecall(w, r)
	case "whoAmI":
		handleWhoAmI(w, r)
	case "getMemories":
		handleGetMemories(w, r)
	case "deleteMemory":
		handleDeleteMemory(w, r)
	case "updateMemory":
		handleUpdateMemory(w, r)
	case "getMemory":
		handleGetMemory(w, r)
	case "listEntities":
		handleListEntities(w, r)
	case "createEntity":
		handleCreateEntity(w, r)
	case "createRelation":
		handleCreateRelation(w, r)
	case "getEntityRelations":
		handleGetEntityRelations(w, r)
	case "addFeedback":
		handleAddFeedback(w, r)
	case "getMemoryHistory":
		handleGetMemoryHistory(w, r)
	case "createSession":
		handleCreateSession(w, r)
	case "getContext":
		handleGetContext(w, r)
	case "createSkill":
		handleCreateSkill(w, r)
	case "listSkills":
		handleListSkills(w, r)
	case "temporalSearch":
		handleTemporalSearch(w, r)
	case "getCompressionStats":
		handleGetCompressionStats(w, r)
	case "setTierPolicy":
		handleSetTierPolicy(w, r)
	case "getProvenance":
		handleGetProvenance(w, r)
	case "createConcept":
		handleCreateConcept(w, r)
	case "linkToConcept":
		handleLinkToConcept(w, r)
	case "setReminder":
		handleSetReminder(w, r)
	case "checkSafety":
		handleCheckSafety(w, r)
	default:
		http.Error(w, "Unknown method: "+method, http.StatusBadRequest)
	}
}
