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
	"syscall"
	"time"

	"agent-memory/internal/connectors"
)

var (
	port               = flag.String("port", "8083", "Port to listen on")
	memoryAPIURL       = flag.String("memory-api", "http://localhost:8081", "Memory API URL")
	notionClientID     = flag.String("notion-client-id", "", "Notion OAuth client ID")
	notionClientSecret = flag.String("notion-client-secret", "", "Notion OAuth client secret")
	githubToken        = flag.String("github-token", "", "GitHub access token")

	// Slack
	slackClientID      = flag.String("slack-client-id", "", "Slack OAuth client ID")
	slackClientSecret  = flag.String("slack-client-secret", "", "Slack OAuth client secret")
	slackSigningSecret = flag.String("slack-signing-secret", "", "Slack signing secret for webhook verification")
	slackRedirectURI   = flag.String("slack-redirect-uri", "", "Slack OAuth redirect URI")

	// Google Drive
	gdriveClientID = flag.String("gdrive-client-id", "", "Google Drive OAuth client ID")
)

type ConnectorsServer struct {
	memoryAPIURL string
	httpServer   *http.Server
}

type Connection struct {
	ID        string    `json:"id"`
	Provider  string    `json:"provider"`
	Status    string    `json:"status"`
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

	// Slack
	mux.HandleFunc("/connectors/slack/webhook", handleSlackWebhook)
	mux.HandleFunc("/connectors/slack/oauth", handleSlackOAuthCallback)

	// Google Drive
	mux.HandleFunc("/connectors/gdrive/oauth", handleGDriveOAuth)
	mux.HandleFunc("/connectors/gdrive/sync", handleGDriveSync)

	// Web Crawler
	mux.HandleFunc("/connectors/crawler", handleCrawlerJob)

	// Status
	mux.HandleFunc("/connectors/status", handleConnectorStatus)
	mux.HandleFunc("/connectors/sync", handleConnectorSync)

	// Health
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/ready", handleReady)

	httpServer := &http.Server{
		Addr:         ":" + *port,
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

func envOrDefault(envKey, fallback string) string {
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	return fallback
}

func applyEnvOverrides() {
	if v := os.Getenv("PORT"); v != "" {
		*port = v
	}
	*memoryAPIURL = envOrDefault("MEMORY_API_URL", *memoryAPIURL)
}

func main() {
	flag.Parse()
	applyEnvOverrides()

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
			RedirectURI string `json:"redirect_uri"`
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

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	secret := os.Getenv("GITHUB_WEBHOOK_SECRET")
	client := connectors.NewGitHubClientWithWebhook(*githubToken, secret)
	event, err := client.HandleWebhook(body, r.Header.Get("X-Hub-Signature-256"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if eventType := r.Header.Get("X-GitHub-Event"); eventType != "" {
		event.Type = eventType
	}
	if action, ok := parseGitHubAction(body); ok {
		event.Action = action
	}

	processed := event.ProcessEvent()

	// Store as memory in Memory API
	_, storeErr := callMemoryAPI("/memories", map[string]interface{}{
		"content": processed,
		"metadata": map[string]interface{}{
			"source": "github-webhook",
			"repo":   event.Repo.Name,
			"owner":  event.Repo.Owner.Login,
		},
	})

	if storeErr != nil {
		log.Printf("Failed to store webhook memory: %v", storeErr)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "processed"}`))
}

// Slack Handlers

func handleSlackWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	// Slack sends the signature in X-Slack-Signature header
	signature := r.Header.Get("X-Slack-Signature")

	slackClient := connectors.NewSlackClient(
		*slackClientID,
		*slackClientSecret,
		"", // access token not needed for webhook verification
		*slackSigningSecret,
		*slackRedirectURI,
	)

	event, err := slackClient.HandleWebhook(body, r.Header.Get("X-Slack-Request-Timestamp"), signature)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Respond to Slack URL verification challenge
	if event.Type == "url_verification" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"challenge": event.Challenge})
		return
	}

	// Store the event as a memory
	content := fmt.Sprintf("[Slack] %s in channel %s: %s", event.User, event.Channel, event.Text)
	_, storeErr := callMemoryAPI("/memories", map[string]interface{}{
		"content": content,
		"metadata": map[string]interface{}{
			"source":   "slack-webhook",
			"channel":  event.Channel,
			"user":     event.User,
			"event_ts": event.EventTS,
		},
	})
	if storeErr != nil {
		log.Printf("Failed to store Slack webhook memory: %v", storeErr)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "processed"}`))
}

func handleSlackOAuthCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Missing authorization code", http.StatusBadRequest)
		return
	}

	slackClient := connectors.NewSlackClient(
		*slackClientID,
		*slackClientSecret,
		"",
		*slackSigningSecret,
		*slackRedirectURI,
	)

	conn, err := slackClient.HandleOAuthCallback(code)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(conn)
}

// Google Drive Handlers

func handleGDriveOAuth(w http.ResponseWriter, r *http.Request) {
	// Accept an access token via query param (after the OAuth redirect from Google)
	accessToken := r.URL.Query().Get("access_token")
	if accessToken == "" {
		// No token yet — return the connection info needed to start the OAuth flow
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"client_id": *gdriveClientID,
			"status":    "oauth_required",
			"message":   "Provide access_token query parameter after completing Google OAuth",
		})
		return
	}

	// Token provided — create a client and verify by listing files
	gdriveClient := connectors.NewGoogleDriveClient(*gdriveClientID, accessToken)
	files := gdriveClient.ListFiles()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "connected",
		"files_found": len(files),
	})
}

func handleGDriveSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var params struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if params.AccessToken == "" {
		http.Error(w, "access_token is required", http.StatusBadRequest)
		return
	}

	gdriveClient := connectors.NewGoogleDriveClient(*gdriveClientID, params.AccessToken)
	files := gdriveClient.ListFiles()

	synced := 0
	for _, f := range files {
		content := fmt.Sprintf("[Google Drive] File: %s (type: %s, id: %s)", f.Name, f.MimeType, f.ID)
		_, err := callMemoryAPI("/memories", map[string]interface{}{
			"content": content,
			"metadata": map[string]interface{}{
				"source":    "gdrive-sync",
				"file_id":   f.ID,
				"file_name": f.Name,
				"mime_type": f.MimeType,
				"link":      f.Link,
			},
		})
		if err != nil {
			log.Printf("Failed to store GDrive file memory for %s: %v", f.Name, err)
			continue
		}
		synced++
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "completed",
		"total_files": len(files),
		"synced":      synced,
	})
}

// Web Crawler Handlers

func handleCrawlerJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var params struct {
		StartURL string `json:"startUrl"`
		MaxDepth int    `json:"maxDepth,omitempty"`
		MaxPages int    `json:"maxPages,omitempty"`
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
			"status":     "ready",
		},
		"slack": map[string]interface{}{
			"configured": *slackClientID != "",
			"status":     "ready",
		},
		"gdrive": map[string]interface{}{
			"configured": *gdriveClientID != "",
			"status":     "ready",
		},
		"crawler": map[string]interface{}{
			"configured": true,
			"status":     "ready",
		},
	}

	json.NewEncoder(w).Encode(status)
}

func handleConnectorSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Connector string                 `json:"connector"`
		Config    map[string]interface{} `json:"config"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Connector == "" {
		http.Error(w, "connector field is required", http.StatusBadRequest)
		return
	}

	switch req.Connector {
	case "notion":
		accessToken, _ := req.Config["access_token"].(string)
		if accessToken == "" {
			http.Error(w, "notion sync requires access_token in config", http.StatusBadRequest)
			return
		}
		limitFloat, _ := req.Config["limit"].(float64)
		notionClient := connectors.NewNotionClient(*notionClientID, *notionClientSecret, accessToken)
		pages, err := notionClient.SyncPages(r.Context(), int(limitFloat))
		if err != nil {
			http.Error(w, fmt.Sprintf("notion sync failed: %v", err), http.StatusInternalServerError)
			return
		}
		synced := 0
		for _, page := range pages {
			content := notionClient.ConvertToMemory(&page)
			_, storeErr := callMemoryAPI("/memories", map[string]interface{}{
				"content": content,
				"metadata": map[string]interface{}{
					"source":  "notion-sync",
					"page_id": page.ID,
					"title":   page.Title,
				},
			})
			if storeErr != nil {
				log.Printf("Failed to store Notion page %s: %v", page.ID, storeErr)
				continue
			}
			synced++
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":      "completed",
			"connector":   "notion",
			"total_pages": len(pages),
			"synced":      synced,
		})

	case "github":
		owner, _ := req.Config["owner"].(string)
		repo, _ := req.Config["repo"].(string)
		if owner == "" || repo == "" {
			http.Error(w, "github sync requires owner and repo in config", http.StatusBadRequest)
			return
		}
		token, _ := req.Config["token"].(string)
		if token == "" {
			token = *githubToken
		}
		githubClient := connectors.NewGitHubClient(token)
		info, err := githubClient.GetRepoInfo(r.Context(), owner, repo)
		if err != nil {
			http.Error(w, fmt.Sprintf("github sync failed: %v", err), http.StatusInternalServerError)
			return
		}
		description, _ := info["description"].(string)
		content := fmt.Sprintf("[GitHub] Repository %s/%s: %s", owner, repo, description)
		_, storeErr := callMemoryAPI("/memories", map[string]interface{}{
			"content": content,
			"metadata": map[string]interface{}{
				"source": "github-sync",
				"owner":  owner,
				"repo":   repo,
			},
		})
		if storeErr != nil {
			log.Printf("Failed to store GitHub repo memory: %v", storeErr)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "completed",
			"connector": "github",
			"synced":    1,
		})

	case "slack":
		accessToken, _ := req.Config["access_token"].(string)
		channelID, _ := req.Config["channel_id"].(string)
		if accessToken == "" {
			http.Error(w, "slack sync requires access_token in config", http.StatusBadRequest)
			return
		}
		slackClient := connectors.NewSlackClient(
			*slackClientID,
			*slackClientSecret,
			accessToken,
			*slackSigningSecret,
			*slackRedirectURI,
		)
		var messages []connectors.SlackMessage
		var syncErr error
		if channelID != "" {
			messages, syncErr = slackClient.SyncMessages(r.Context(), channelID)
		} else {
			// Sync all readable channels
			channels, err := slackClient.GetConversations(r.Context(), 100)
			if err != nil {
				http.Error(w, fmt.Sprintf("slack sync failed listing channels: %v", err), http.StatusInternalServerError)
				return
			}
			for _, ch := range channels {
				chMessages, err := slackClient.SyncMessages(r.Context(), ch.ID)
				if err != nil {
					log.Printf("Failed to sync Slack channel %s: %v", ch.ID, err)
					continue
				}
				messages = append(messages, chMessages...)
			}
		}
		if syncErr != nil {
			http.Error(w, fmt.Sprintf("slack sync failed: %v", syncErr), http.StatusInternalServerError)
			return
		}
		synced := 0
		for _, msg := range messages {
			content := fmt.Sprintf("[Slack] %s in channel %s: %s", msg.User, msg.Channel, msg.Content)
			_, storeErr := callMemoryAPI("/memories", map[string]interface{}{
				"content": content,
				"metadata": map[string]interface{}{
					"source":  "slack-sync",
					"channel": msg.Channel,
					"user":    msg.User,
					"ts":      msg.TS,
				},
			})
			if storeErr != nil {
				log.Printf("Failed to store Slack message: %v", storeErr)
				continue
			}
			synced++
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":         "completed",
			"connector":      "slack",
			"total_messages": len(messages),
			"synced":         synced,
		})

	case "gdrive":
		accessToken, _ := req.Config["access_token"].(string)
		if accessToken == "" {
			http.Error(w, "gdrive sync requires access_token in config", http.StatusBadRequest)
			return
		}
		gdriveClient := connectors.NewGoogleDriveClient(*gdriveClientID, accessToken)
		files := gdriveClient.ListFiles()
		synced := 0
		for _, f := range files {
			content := fmt.Sprintf("[Google Drive] File: %s (type: %s, id: %s)", f.Name, f.MimeType, f.ID)
			_, storeErr := callMemoryAPI("/memories", map[string]interface{}{
				"content": content,
				"metadata": map[string]interface{}{
					"source":    "gdrive-sync",
					"file_id":   f.ID,
					"file_name": f.Name,
					"mime_type": f.MimeType,
					"link":      f.Link,
				},
			})
			if storeErr != nil {
				log.Printf("Failed to store GDrive file memory for %s: %v", f.Name, storeErr)
				continue
			}
			synced++
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":      "completed",
			"connector":   "gdrive",
			"total_files": len(files),
			"synced":      synced,
		})

	case "crawler":
		startURL, _ := req.Config["start_url"].(string)
		if startURL == "" {
			http.Error(w, "crawler sync requires start_url in config", http.StatusBadRequest)
			return
		}
		maxDepthFloat, _ := req.Config["max_depth"].(float64)
		maxPagesFloat, _ := req.Config["max_pages"].(float64)
		crawler := connectors.NewWebCrawler()
		result, err := crawler.CrawlWithConfig(r.Context(), startURL, &connectors.CrawlConfig{
			MaxDepth: int(maxDepthFloat),
			MaxPages: int(maxPagesFloat),
		})
		if err != nil {
			http.Error(w, fmt.Sprintf("crawler sync failed: %v", err), http.StatusInternalServerError)
			return
		}
		synced := 0
		for _, page := range result.Pages {
			content := crawler.ConvertToMemory(&page)
			_, storeErr := callMemoryAPI("/memories", map[string]interface{}{
				"content": content,
				"metadata": map[string]interface{}{
					"source": "crawler-sync",
					"url":    page.URL,
				},
			})
			if storeErr != nil {
				log.Printf("Failed to store crawler page memory for %s: %v", page.URL, storeErr)
				continue
			}
			synced++
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":      "completed",
			"connector":   "crawler",
			"total_pages": len(result.Pages),
			"synced":      synced,
		})

	default:
		http.Error(w, fmt.Sprintf("unknown connector: %s", req.Connector), http.StatusBadRequest)
	}
}

// ==================== Helpers ====================

func callMemoryAPI(path string, payload interface{}) ([]byte, error) {
	url := *memoryAPIURL + path

	marshal, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(marshal))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	resultBytes, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("marshal result: %w", err)
	}
	return resultBytes, nil
}
