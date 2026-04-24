package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"agent-memory/internal/connectors"
)

var (
	port          = flag.String("port", "8083", "Port to listen on")
	memoryAPIURL  = flag.String("memory-api", "http://localhost:8081", "Memory API URL")
	notionClientID     = flag.String("notion-client-id", "", "Notion OAuth client ID")
	notionClientSecret = flag.String("notion-client-secret", "", "Notion OAuth client secret")
	githubToken       = flag.String("github-token", "", "GitHub access token")
)

type ConnectorsServer struct {
	memoryAPIURL string
	httpServer   *http.Server
}

type Connection struct {
	ID        string    `json:"id"`
	Provider string    `json:"provider"`
	Status   string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

func NewConnectorsServer() *ConnectorsServer {
	mux := http.NewServeMux()
	
	// Notion
	mux.HandleFunc("/connectors/notion", handleNotionConnection)
	mux.HandleFunc("/connectors/notion/oauth", handleNotionOAuthCallback)
	
	// GitHub
	mux.HandleFunc("/connectors/github", handleGitHubConnection)
	mux.HandleFunc("/connectors/github/webhook", handleGitHubWebhook)
	
	// Web Crawler
	mux.HandleFunc("/connectors/crawler", handleCrawlerJob)
	
	// Status
	mux.HandleFunc("/connectors/status", handleConnectorStatus)
	mux.HandleFunc("/connectors/sync", handleConnectorSync)
	
	// Health
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/ready", handleReady)

	httpServer := &http.Server{
		Addr:         *port,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return &ConnectorsServer{
		memoryAPIURL: *memoryAPIURL,
		httpServer:   httpServer,
	}
}

func (s *ConnectorsServer) Start() error {
	log.Printf("Connectors Server starting on %s", *port)
	log.Printf("Memory API endpoint: %s", s.memoryAPIURL)
	
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("Connectors server: %w", err)
	}
	return nil
}

func (s *ConnectorsServer) Stop(ctx context.Context) error {
	log.Println("Shutting down Connectors Server...")
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	return nil
}

func main() {
	flag.Parse()

	server := NewConnectorsServer()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Graceful shutdown handled via signal
	go func() {
		<-sigChan
		log.Println("Received shutdown signal")
		server.Stop(context.TODO())
	}()

	if err := server.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// ==================== Handlers ====================

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "ok", "service": "connectors"}`))
}

func handleReady(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "ready"}`))
}

// Notion Handlers

func handleNotionConnection(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var params struct {
			AccessToken string `json:"access_token"`
			RedirectURI  string `json:"redirect_uri"`
		}
		if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		// Create connection
		notionClient := connectors.NewNotionClient(
			*notionClientID,
			*notionClientSecret,
			params.AccessToken,
		)

		conn := notionClient.CreateConnection("Notion Workspace", params.RedirectURI)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(conn)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func handleNotionOAuthCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	redirectURI := r.URL.Query().Get("redirect_uri")

	if code == "" {
		http.Error(w, "Missing authorization code", http.StatusBadRequest)
		return
	}

	notionClient := connectors.NewNotionClient(*notionClientID, *notionClientSecret, "")
	resp, err := notionClient.HandleOAuthCallback(connectors.NotionOAuthCallback{
		Code:        code,
		RedirectURI: redirectURI,
	})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// GitHub Handlers

func handleGitHubConnection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var params struct {
		Owner string `json:"owner"`
		Repo  string `json:"repo"`
	}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	githubClient := connectors.NewGitHubClient(*githubToken)
	info, err := githubClient.GetRepoInfo(r.Context(), params.Owner, params.Repo)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

func handleGitHubWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	// Process webhook
	var event connectors.GitHubEvent
	jsonData, _ := json.Marshal(payload)
	json.Unmarshal(jsonData, &event)

	processed := event.ProcessEvent()

	// Store as memory in Memory API
	_, err := callMemoryAPI("/memories", map[string]interface{}{
		"content":  processed,
		"metadata": map[string]interface{}{
			"source": "github-webhook",
			"repo":   event.Repo.Name,
			"owner":  event.Repo.Owner.Login,
		},
	})

	if err != nil {
		log.Printf("Failed to store webhook memory: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "processed"}`))
}

// Web Crawler Handlers

func handleCrawlerJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var params struct {
		StartURL string `json:"startUrl"`
		MaxDepth int   `json:"maxDepth,omitempty"`
		MaxPages int   `json:"maxPages,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	crawler := connectors.NewWebCrawler()
	result, err := crawler.CrawlWithConfig(r.Context(), params.StartURL, &connectors.CrawlConfig{
		MaxDepth: params.MaxDepth,
		MaxPages: params.MaxPages,
	})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Store each page as memory
	for _, page := range result.Pages {
		content := crawler.ConvertToMemory(&page)
		_, err := callMemoryAPI("/memories", map[string]interface{}{
			"content": content,
			"metadata": map[string]interface{}{
				"source": "web-crawler",
				"url":    page.URL,
			},
		})
		if err != nil {
			log.Printf("Failed to store memory for %s: %v", page.URL, err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result.Summary)
}

// Status Handlers

func handleConnectorStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	status := map[string]interface{}{
		"notion": map[string]interface{}{
			"configured": *notionClientID != "",
			"status":     "ready",
		},
		"github": map[string]interface{}{
			"configured": *githubToken != "",
			"status":    "ready",
		},
		"crawler": map[string]interface{}{
			"configured": true,
			"status":    "ready",
		},
	}
	
	json.NewEncoder(w).Encode(status)
}

func handleConnectorSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Manual sync trigger - for now just return success
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "sync initiated"}`))
}

// ==================== Helpers ====================

func callMemoryAPI(path string, payload interface{}) ([]byte, error) {
	url := *memoryAPIURL + path
	
	marshal, _ := json.Marshal(payload)
	_ = string(marshal) // payload for logging

	req, _ := http.NewRequest("POST", url, bytes.NewReader(marshal))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		ID string `json:"id"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	marshal, _ = json.Marshal(result)
	return marshal, nil
}