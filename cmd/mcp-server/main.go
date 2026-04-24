package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

var (
	port           = flag.String("port", "8082", "Port to listen on")
	memoryAPIURL   = flag.String("memory-api", "http://localhost:8081", "Memory API URL")
	enableOAuth  = flag.Bool("oauth", false, "Enable OAuth authentication")
	oauthSecret   = flag.String("oauth-secret", "default-secret", "OAuth secret key")
)

type MCPServer struct {
	memoryAPIURL string
	httpServer  *http.Server
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
	
	// Health
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/ready", handleReady)

	httpServer := &http.Server{
		Addr:         *port,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return &MCPServer{
		memoryAPIURL: *memoryAPIURL,
		httpServer:  httpServer,
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
	if r.Method == http.MethodGet {
		// Return tools list
		tools := []map[string]interface{}{
			{"name": "addMemory", "description": "Add a memory to the store"},
			{"name": "recall", "description": "Search memories"},
			{"name": "search", "description": "Search memories (alias for recall)"},
			{"name": "whoAmI", "description": "Get current user info"},
			{"name": "getMemories", "description": "List all memories"},
			{"name": "deleteMemory", "description": "Delete a memory"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"tools": tools})
		return
	}

	if r.Method == http.MethodPost {
		// Handle tool call
		var req struct {
			Method string `json:"method"`
			Params map[string]interface{} `json:"params,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		// Route to appropriate handler
		switch req.Method {
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
		default:
			http.Error(w, "Unknown method", http.StatusBadRequest)
		}
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"userId": "default", "role": "user", "status": "active"}`))
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
	var params struct {
		MemoryID string `json:"memoryId"`
	}
	json.NewDecoder(r.Body).Decode(&params)

	// Call Memory API
	resp, err := callMemoryAPI("/memories/"+params.MemoryID, nil)
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

func handleOAuthAuthorize(w http.ResponseWriter, r *http.Request) {
	// Simplified OAuth authorize
	state := r.URL.Query().Get("state")
	redirectURI := r.URL.Query().Get("redirect_uri")
	
	// In production, redirect to login page
	if redirectURI != "" {
		http.Redirect(w, r, redirectURI+"?code=mock&state="+state, http.StatusFound)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"auth_url": "/oauth/authorize",
		"state": state,
	})
}

func handleOAuthToken(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	// Simplified token response
	json.NewEncoder(w).Encode(map[string]interface{}{
		"access_token":  "mock-token",
		"token_type":    "Bearer",
		"expires_in":   3600,
		"refresh_token": "mock-refresh",
	})
}

// ==================== Helpers ====================

func callMemoryAPI(path string, payload interface{}) ([]byte, error) {
	url := *memoryAPIURL + path
	
	var body *strings.Reader
	if payload != nil {
		b, _ := json.Marshal(payload)
		body = strings.NewReader(string(b))
	} else {
		body = strings.NewReader("")
	}

	req, _ := http.NewRequest("POST", url, body)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	buf := make([]byte, resp.ContentLength+100)
	resp.Body.Read(buf)
	return buf, nil
}