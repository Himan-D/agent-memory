package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"agent-memory/internal/connectors"
	"agent-memory/internal/memory/types"
)

func (s *APIServer) connectorStatusHandler(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"notion": map[string]interface{}{
			"configured": os.Getenv("NOTION_CLIENT_ID") != "",
			"status":     "ready",
		},
		"github": map[string]interface{}{
			"configured": os.Getenv("GITHUB_TOKEN") != "",
			"status":     "ready",
		},
		"slack": map[string]interface{}{
			"configured": os.Getenv("SLACK_CLIENT_ID") != "",
			"status":     "ready",
		},
		"gdrive": map[string]interface{}{
			"configured": os.Getenv("GDRIVE_CLIENT_ID") != "",
			"status":     "ready",
		},
		"crawler": map[string]interface{}{
			"configured": true,
			"status":     "ready",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (s *APIServer) connectorSyncHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Connector string                 `json:"connector"`
		Config    map[string]interface{} `json:"config"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Connector == "" {
		jsonError(w, "connector field is required", http.StatusBadRequest)
		return
	}

	switch req.Connector {
	case "notion":
		s.syncNotionConnector(w, r, req.Config)
	case "github":
		s.syncGitHubConnector(w, r, req.Config)
	case "slack":
		s.syncSlackConnector(w, r, req.Config)
	case "gdrive":
		s.syncGDriveConnector(w, r, req.Config)
	case "crawler":
		s.syncCrawlerConnector(w, r, req.Config)
	default:
		jsonError(w, "unknown connector: "+req.Connector, http.StatusBadRequest)
	}
}

func (s *APIServer) connectorCrawlerHandler(w http.ResponseWriter, r *http.Request) {
	var params struct {
		StartURL string `json:"startUrl"`
		MaxDepth int    `json:"maxDepth,omitempty"`
		MaxPages int    `json:"maxPages,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if params.StartURL == "" {
		jsonError(w, "startUrl is required", http.StatusBadRequest)
		return
	}

	crawler := connectors.NewWebCrawler()
	result, err := crawler.CrawlWithConfig(r.Context(), params.StartURL, &connectors.CrawlConfig{
		MaxDepth: params.MaxDepth,
		MaxPages: params.MaxPages,
	})
	if err != nil {
		safeHTTPError(w, r, fmt.Errorf("crawler: %w", err), http.StatusInternalServerError)
		return
	}

	synced := 0
	for _, page := range result.Pages {
		content := crawler.ConvertToMemory(&page)
		if _, err := s.storeConnectorMemory(r, content, map[string]interface{}{
			"source": "web-crawler",
			"url":    page.URL,
		}); err != nil {
			log.Printf("Failed to store memory for %s: %v", page.URL, err)
			continue
		}
		synced++
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "completed",
		"total_pages": len(result.Pages),
		"synced":      synced,
		"summary":     result.Summary,
	})
}

func (s *APIServer) storeConnectorMemory(r *http.Request, content string, metadata map[string]interface{}) (*types.Memory, error) {
	tenantID := getTenantID(r)
	if tenantID == "" {
		tenantID = "default"
	}
	mem := &types.Memory{
		Content:  content,
		TenantID: tenantID,
		OrgID:    "default",
		Type:     types.MemoryTypeUser,
		Metadata: metadata,
	}
	return s.memSvc.CreateMemory(r.Context(), mem)
}

func (s *APIServer) syncNotionConnector(w http.ResponseWriter, r *http.Request, config map[string]interface{}) {
	accessToken, _ := config["access_token"].(string)
	if accessToken == "" {
		jsonError(w, "notion sync requires access_token in config", http.StatusBadRequest)
		return
	}
	limitFloat, _ := config["limit"].(float64)
	client := connectors.NewNotionClient(
		os.Getenv("NOTION_CLIENT_ID"),
		os.Getenv("NOTION_CLIENT_SECRET"),
		accessToken,
	)
	pages, err := client.SyncPages(r.Context(), int(limitFloat))
	if err != nil {
		safeHTTPError(w, r, fmt.Errorf("notion sync: %w", err), http.StatusInternalServerError)
		return
	}

	synced := 0
	for _, page := range pages {
		content := client.ConvertToMemory(&page)
		if _, err := s.storeConnectorMemory(r, content, map[string]interface{}{
			"source":  "notion-sync",
			"page_id": page.ID,
			"title":   page.Title,
		}); err != nil {
			log.Printf("Failed to store Notion page %s: %v", page.ID, err)
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
}

func (s *APIServer) syncGitHubConnector(w http.ResponseWriter, r *http.Request, config map[string]interface{}) {
	owner, _ := config["owner"].(string)
	repo, _ := config["repo"].(string)
	if owner == "" || repo == "" {
		jsonError(w, "github sync requires owner and repo in config", http.StatusBadRequest)
		return
	}
	token, _ := config["token"].(string)
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}
	client := connectors.NewGitHubClient(token)
	info, err := client.GetRepoInfo(r.Context(), owner, repo)
	if err != nil {
		safeHTTPError(w, r, fmt.Errorf("github sync: %w", err), http.StatusInternalServerError)
		return
	}
	description, _ := info["description"].(string)
	content := fmt.Sprintf("[GitHub] Repository %s/%s: %s", owner, repo, description)
	if _, err := s.storeConnectorMemory(r, content, map[string]interface{}{
		"source": "github-sync",
		"owner":  owner,
		"repo":   repo,
	}); err != nil {
		safeHTTPError(w, r, fmt.Errorf("store github memory: %w", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "completed",
		"connector": "github",
		"synced":    1,
	})
}

func (s *APIServer) syncSlackConnector(w http.ResponseWriter, r *http.Request, config map[string]interface{}) {
	accessToken, _ := config["access_token"].(string)
	channelID, _ := config["channel_id"].(string)
	if accessToken == "" {
		jsonError(w, "slack sync requires access_token in config", http.StatusBadRequest)
		return
	}
	client := connectors.NewSlackClient(
		os.Getenv("SLACK_CLIENT_ID"),
		os.Getenv("SLACK_CLIENT_SECRET"),
		accessToken,
		os.Getenv("SLACK_SIGNING_SECRET"),
		os.Getenv("SLACK_REDIRECT_URI"),
	)

	var messages []connectors.SlackMessage
	if channelID != "" {
		var err error
		messages, err = client.SyncMessages(r.Context(), channelID)
		if err != nil {
			safeHTTPError(w, r, fmt.Errorf("slack sync: %w", err), http.StatusInternalServerError)
			return
		}
	} else {
		channels, err := client.GetConversations(r.Context(), 100)
		if err != nil {
			safeHTTPError(w, r, fmt.Errorf("slack list channels: %w", err), http.StatusInternalServerError)
			return
		}
		for _, ch := range channels {
			chMessages, err := client.SyncMessages(r.Context(), ch.ID)
			if err != nil {
				log.Printf("Failed to sync Slack channel %s: %v", ch.ID, err)
				continue
			}
			messages = append(messages, chMessages...)
		}
	}

	synced := 0
	for _, msg := range messages {
		content := fmt.Sprintf("[Slack] %s in channel %s: %s", msg.User, msg.Channel, msg.Content)
		if _, err := s.storeConnectorMemory(r, content, map[string]interface{}{
			"source":  "slack-sync",
			"channel": msg.Channel,
			"user":    msg.User,
			"ts":      msg.TS,
		}); err != nil {
			log.Printf("Failed to store Slack message: %v", err)
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
}

func (s *APIServer) syncGDriveConnector(w http.ResponseWriter, r *http.Request, config map[string]interface{}) {
	accessToken, _ := config["access_token"].(string)
	if accessToken == "" {
		jsonError(w, "gdrive sync requires access_token in config", http.StatusBadRequest)
		return
	}
	client := connectors.NewGoogleDriveClient(os.Getenv("GDRIVE_CLIENT_ID"), accessToken)
	files := client.ListFiles()
	synced := 0
	for _, f := range files {
		content := fmt.Sprintf("[Google Drive] File: %s (type: %s, id: %s)", f.Name, f.MimeType, f.ID)
		if _, err := s.storeConnectorMemory(r, content, map[string]interface{}{
			"source":    "gdrive-sync",
			"file_id":   f.ID,
			"file_name": f.Name,
			"mime_type": f.MimeType,
			"link":      f.Link,
		}); err != nil {
			log.Printf("Failed to store GDrive file memory for %s: %v", f.Name, err)
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
}

func (s *APIServer) syncCrawlerConnector(w http.ResponseWriter, r *http.Request, config map[string]interface{}) {
	startURL, _ := config["start_url"].(string)
	if startURL == "" {
		jsonError(w, "crawler sync requires start_url in config", http.StatusBadRequest)
		return
	}
	maxDepth, _ := config["max_depth"].(float64)
	maxPages, _ := config["max_pages"].(float64)

	crawler := connectors.NewWebCrawler()
	result, err := crawler.CrawlWithConfig(r.Context(), startURL, &connectors.CrawlConfig{
		MaxDepth: int(maxDepth),
		MaxPages: int(maxPages),
	})
	if err != nil {
		safeHTTPError(w, r, fmt.Errorf("crawler sync: %w", err), http.StatusInternalServerError)
		return
	}

	synced := 0
	for _, page := range result.Pages {
		content := crawler.ConvertToMemory(&page)
		if _, err := s.storeConnectorMemory(r, content, map[string]interface{}{
			"source": "web-crawler",
			"url":    page.URL,
		}); err != nil {
			log.Printf("Failed to store memory for %s: %v", page.URL, err)
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
}
