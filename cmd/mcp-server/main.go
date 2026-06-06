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
	"syscall"
	"time"
)

var (
	port         = flag.String("port", "8082", "Port to listen on")
	memoryAPIURL = flag.String("memory-api", "http://localhost:8081", "Memory API URL")
	enableOAuth  = flag.Bool("oauth", false, "Enable OAuth authentication")
	oauthSecret  = flag.String("oauth-secret", "default-secret", "OAuth secret key")
)

type MCPServer struct {
	memoryAPIURL string
	httpServer   *http.Server
}

func NewMCPServer() *MCPServer {
	mux := http.NewServeMux()

	// MCP Protocol
	mux.HandleFunc("/mcp", handleMCP)

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

	httpServer := &http.Server{
		Addr:         ":" + *port,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return &MCPServer{
		memoryAPIURL: *memoryAPIURL,
		httpServer:   httpServer,
	}
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

func applyEnvOverrides() {
	if v := os.Getenv("PORT"); v != "" {
		*port = v
	}
	*memoryAPIURL = envOrDefault("MEMORY_API_URL", *memoryAPIURL)
}

func envOrDefault(envKey, fallback string) string {
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	return fallback
}

func main() {
	flag.Parse()
	applyEnvOverrides()

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

// MCP Protocol Handler
func handleMCP(w http.ResponseWriter, r *http.Request) {
	// Handle CORS for MCP protocol
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Tool list endpoint
	if r.URL.Path == "" || r.URL.Path == "/" {
		tools := []map[string]string{
			{"name": "addMemory", "description": "Add a new memory"},
			{"name": "recall", "description": "Search memories"},
			{"name": "search", "description": "Search memories (alias for recall)"},
			{"name": "whoAmI", "description": "Get current user info"},
			{"name": "getMemories", "description": "List all memories"},
			{"name": "deleteMemory", "description": "Delete a memory"},
			{"name": "updateMemory", "description": "Update an existing memory"},
			{"name": "getMemory", "description": "Get a specific memory by ID"},
			{"name": "listEntities", "description": "List knowledge graph entities"},
			{"name": "createEntity", "description": "Create a knowledge graph entity"},
			{"name": "createRelation", "description": "Create a relation between entities"},
			{"name": "getEntityRelations", "description": "Get relations for an entity"},
			{"name": "addFeedback", "description": "Add feedback to a memory"},
			{"name": "getMemoryHistory", "description": "Get modification history for a memory"},
			{"name": "createSession", "description": "Create a new conversation session"},
			{"name": "getContext", "description": "Get conversation context for a session"},
			{"name": "createSkill", "description": "Create a new skill"},
			{"name": "listSkills", "description": "List available skills"},
			{"name": "temporalSearch", "description": "Search memories within a time range"},
			{"name": "getCompressionStats", "description": "Get compression engine statistics"},
			{"name": "setTierPolicy", "description": "Set the memory tier policy"},
			{"name": "getProvenance", "description": "Get provenance chain for a memory"},
			{"name": "createConcept", "description": "Create a concept node in the knowledge graph"},
			{"name": "linkToConcept", "description": "Link a memory or entity to a concept"},
			{"name": "setReminder", "description": "Set a prospective memory reminder on a memory"},
			{"name": "checkSafety", "description": "Check content safety classification"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"tools": tools})
		return
	}

	// Read body once and reuse for method routing
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	if r.Method == http.MethodPost || r.Method == http.MethodGet {
		// Handle tool call
		var req struct {
			Method string                 `json:"method"`
			Params map[string]interface{} `json:"params,omitempty"`
		}
		if err := json.Unmarshal(bodyBytes, &req); err != nil {
			// Try to route by URL path for GET or direct method param
			method := r.URL.Query().Get("method")
			if method == "" {
				method = r.URL.Query().Get("m")
			}

			if method != "" {
				routeByMethod(w, r, method, bodyBytes)
				return
			}
		}

		// Route to appropriate handler
		if req.Method != "" {
			routeByMethod(w, r, req.Method, bodyBytes)
			return
		}

		// Try URL path based routing
		method := strings.TrimPrefix(r.URL.Path, "/mcp/")
		if method != "" && method != r.URL.Path {
			routeByMethod(w, r, method, bodyBytes)
			return
		}

		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func routeByMethod(w http.ResponseWriter, r *http.Request, method string, bodyBytes []byte) {
	// Restore body for handlers that need it
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
