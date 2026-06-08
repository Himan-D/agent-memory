package connections

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"agent-memory/internal/connectors"
	"agent-memory/internal/sources"
)

const (
	ProviderNotion     = "notion"
	ProviderGDrive     = "gdrive"
	ProviderGitHub     = "github"
	ProviderSlack      = "slack"
	ProviderS3         = "s3"
	ProviderWebCrawler = "web_crawler"

	StatusActive        = "active"
	StatusOAuthRequired = "oauth_required"
	StatusNotConfigured = "not_configured"
	StatusSyncing       = "syncing"
	StatusSynced        = "synced"
	StatusError         = "error"
)

var supportedProviders = map[string]bool{
	ProviderNotion:     true,
	ProviderGDrive:     true,
	ProviderGitHub:     true,
	ProviderSlack:      true,
	ProviderS3:         true,
	ProviderWebCrawler: true,
}

type SourceIngestor interface {
	Ingest(ctx context.Context, req sources.IngestRequest) (*sources.IngestResult, error)
	Delete(ctx context.Context, sourceID string) error
}

type Store interface {
	Save(ctx context.Context, connection *Connection) error
	Get(ctx context.Context, id string) (*Connection, error)
	List(ctx context.Context, scope Scope) ([]*Connection, error)
	Delete(ctx context.Context, id string) error
}

type Scope struct {
	UserID string
	OrgID  string
}

type Connection struct {
	ID              string                 `json:"id"`
	Provider        string                 `json:"provider"`
	Status          string                 `json:"status"`
	UserID          string                 `json:"user_id,omitempty"`
	OrgID           string                 `json:"org_id,omitempty"`
	TenantID        string                 `json:"tenant_id,omitempty"`
	Config          map[string]interface{} `json:"config,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	SourceIDs       []string               `json:"source_ids,omitempty"`
	LastSyncedAt    *time.Time             `json:"last_synced_at,omitempty"`
	LastError       string                 `json:"last_error,omitempty"`
	SyncCount       int                    `json:"sync_count"`
	SyncedDocuments int                    `json:"synced_documents"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

type CreateRequest struct {
	UserID   string                 `json:"user_id,omitempty"`
	OrgID    string                 `json:"org_id,omitempty"`
	TenantID string                 `json:"tenant_id,omitempty"`
	Config   map[string]interface{} `json:"config,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type SyncRequest struct {
	Limit     int                    `json:"limit,omitempty"`
	Documents []SeedDocument         `json:"documents,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type SeedDocument struct {
	Title      string                 `json:"title,omitempty"`
	Content    string                 `json:"content,omitempty"`
	URL        string                 `json:"url,omitempty"`
	Type       string                 `json:"type,omitempty"`
	ExternalID string                 `json:"external_id,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

type SyncResult struct {
	Connection *Connection `json:"connection"`
	Status     string      `json:"status"`
	Synced     int         `json:"synced"`
	SourceIDs  []string    `json:"source_ids,omitempty"`
	Error      string      `json:"error,omitempty"`
}

type Service struct {
	store   Store
	sources SourceIngestor
}

func NewService(store Store, sourceIngestor SourceIngestor) *Service {
	if store == nil {
		store = NewInMemoryStore()
	}
	return &Service{store: store, sources: sourceIngestor}
}

func SupportedProviders() []string {
	providers := make([]string, 0, len(supportedProviders))
	for provider := range supportedProviders {
		providers = append(providers, provider)
	}
	sort.Strings(providers)
	return providers
}

func (s *Service) Create(ctx context.Context, provider string, req CreateRequest) (*Connection, error) {
	provider = normalizeProvider(provider)
	if !supportedProviders[provider] {
		return nil, fmt.Errorf("unsupported provider %q", provider)
	}
	now := time.Now().UTC()
	conn := &Connection{
		ID:        uuid.New().String(),
		Provider:  provider,
		Status:    initialStatus(provider, req.Config),
		UserID:    req.UserID,
		OrgID:     req.OrgID,
		TenantID:  req.TenantID,
		Config:    cloneMap(req.Config),
		Metadata:  cloneMap(req.Metadata),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.store.Save(ctx, conn); err != nil {
		return nil, err
	}
	return conn.Sanitized(), nil
}

func (s *Service) Get(ctx context.Context, id string) (*Connection, error) {
	conn, err := s.store.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return conn.Sanitized(), nil
}

func (s *Service) List(ctx context.Context, scope Scope) ([]*Connection, error) {
	conns, err := s.store.List(ctx, scope)
	if err != nil {
		return nil, err
	}
	out := make([]*Connection, 0, len(conns))
	for _, conn := range conns {
		out = append(out, conn.Sanitized())
	}
	return out, nil
}

func (s *Service) Delete(ctx context.Context, id string, deleteDocuments bool) error {
	conn, err := s.store.Get(ctx, id)
	if err != nil {
		return err
	}
	if deleteDocuments && s.sources != nil {
		for _, sourceID := range conn.SourceIDs {
			_ = s.sources.Delete(ctx, sourceID)
		}
	}
	return s.store.Delete(ctx, id)
}

func (s *Service) Sync(ctx context.Context, id string, req SyncRequest) (*SyncResult, error) {
	conn, err := s.store.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if s.sources == nil {
		return nil, fmt.Errorf("source ingestion service is not configured")
	}
	conn.Status = StatusSyncing
	conn.UpdatedAt = time.Now().UTC()
	_ = s.store.Save(ctx, conn)

	docs, err := s.documentsForConnection(ctx, conn, req)
	if err != nil {
		conn.Status = statusForMissingConfig(conn.Provider, conn.Config)
		conn.LastError = err.Error()
		conn.UpdatedAt = time.Now().UTC()
		_ = s.store.Save(ctx, conn)
		return &SyncResult{Connection: conn.Sanitized(), Status: conn.Status, Error: err.Error()}, err
	}

	sourceIDs := make([]string, 0, len(docs))
	for _, doc := range docs {
		result, ingestErr := s.sources.Ingest(ctx, sources.IngestRequest{
			Type:       documentType(doc),
			Content:    doc.Content,
			URL:        doc.URL,
			Title:      doc.Title,
			Provider:   conn.Provider,
			ExternalID: doc.ExternalID,
			UserID:     conn.UserID,
			OrgID:      conn.OrgID,
			Metadata:   mergeMaps(conn.Metadata, req.Metadata, doc.Metadata, map[string]interface{}{"connection_id": conn.ID}),
		})
		if ingestErr != nil {
			conn.Status = StatusError
			conn.LastError = ingestErr.Error()
			conn.UpdatedAt = time.Now().UTC()
			_ = s.store.Save(ctx, conn)
			return &SyncResult{Connection: conn.Sanitized(), Status: conn.Status, Synced: len(sourceIDs), SourceIDs: sourceIDs, Error: ingestErr.Error()}, ingestErr
		}
		sourceIDs = append(sourceIDs, result.SourceID)
	}

	now := time.Now().UTC()
	conn.Status = StatusSynced
	conn.LastSyncedAt = &now
	conn.LastError = ""
	conn.SyncCount++
	conn.SyncedDocuments += len(sourceIDs)
	conn.SourceIDs = append(conn.SourceIDs, sourceIDs...)
	conn.UpdatedAt = now
	if err := s.store.Save(ctx, conn); err != nil {
		return nil, err
	}
	return &SyncResult{Connection: conn.Sanitized(), Status: conn.Status, Synced: len(sourceIDs), SourceIDs: sourceIDs}, nil
}

func (s *Service) documentsForConnection(ctx context.Context, conn *Connection, req SyncRequest) ([]SeedDocument, error) {
	if len(req.Documents) > 0 {
		return req.Documents, nil
	}
	switch conn.Provider {
	case ProviderWebCrawler:
		startURL := stringValue(conn.Config, "start_url", "url")
		if startURL == "" {
			return nil, fmt.Errorf("web crawler requires start_url")
		}
		return []SeedDocument{{Title: hostTitle(startURL), URL: startURL, Type: "url", ExternalID: startURL}}, nil
	case ProviderNotion:
		token := stringValue(conn.Config, "access_token", "token")
		if token == "" {
			return nil, fmt.Errorf("notion access_token is required")
		}
		limit := limitOrDefault(req.Limit, 10)
		pages, err := connectors.NewNotionClient("", "", token).SyncPages(ctx, limit)
		if err != nil {
			return nil, err
		}
		docs := make([]SeedDocument, 0, len(pages))
		for _, page := range pages {
			docs = append(docs, SeedDocument{
				Title:      page.Title,
				Content:    page.Content,
				URL:        page.URL,
				Type:       "text",
				ExternalID: page.ID,
				Metadata:   map[string]interface{}{"notion_updated_at": page.LastEdited},
			})
		}
		return docs, nil
	case ProviderGitHub:
		owner := stringValue(conn.Config, "owner")
		repo := stringValue(conn.Config, "repo")
		if owner == "" || repo == "" {
			return nil, fmt.Errorf("github requires owner and repo")
		}
		repoURL := fmt.Sprintf("https://github.com/%s/%s", owner, repo)
		content := fmt.Sprintf("GitHub repository %s/%s connected to Hystersis memory. Sync issues, pull requests, and repository notes through this connection.", owner, repo)
		return []SeedDocument{{Title: fmt.Sprintf("%s/%s", owner, repo), Content: content, URL: repoURL, Type: "text", ExternalID: repoURL}}, nil
	case ProviderGDrive:
		return nil, fmt.Errorf("google drive OAuth is required before sync")
	case ProviderSlack:
		return nil, fmt.Errorf("slack OAuth is required before sync")
	case ProviderS3:
		if stringValue(conn.Config, "bucket") == "" {
			return nil, fmt.Errorf("s3 requires bucket")
		}
		return nil, fmt.Errorf("s3 object listing credentials are required before sync")
	default:
		return nil, fmt.Errorf("unsupported provider %q", conn.Provider)
	}
}

func (c *Connection) Sanitized() *Connection {
	if c == nil {
		return nil
	}
	cp := *c
	cp.Config = redactMap(c.Config)
	cp.Metadata = cloneMap(c.Metadata)
	cp.SourceIDs = append([]string(nil), c.SourceIDs...)
	return &cp
}

type InMemoryStore struct {
	mu          sync.RWMutex
	connections map[string]*Connection
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{connections: map[string]*Connection{}}
}

func (s *InMemoryStore) Save(ctx context.Context, connection *Connection) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *connection
	cp.Config = cloneMap(connection.Config)
	cp.Metadata = cloneMap(connection.Metadata)
	cp.SourceIDs = append([]string(nil), connection.SourceIDs...)
	s.connections[connection.ID] = &cp
	return nil
}

func (s *InMemoryStore) Get(ctx context.Context, id string) (*Connection, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()
	conn, ok := s.connections[id]
	if !ok {
		return nil, fmt.Errorf("connection not found")
	}
	cp := *conn
	cp.Config = cloneMap(conn.Config)
	cp.Metadata = cloneMap(conn.Metadata)
	cp.SourceIDs = append([]string(nil), conn.SourceIDs...)
	return &cp, nil
}

func (s *InMemoryStore) List(ctx context.Context, scope Scope) ([]*Connection, error) {
	_ = ctx
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Connection, 0, len(s.connections))
	for _, conn := range s.connections {
		if scope.UserID != "" && conn.UserID != scope.UserID {
			continue
		}
		if scope.OrgID != "" && conn.OrgID != scope.OrgID {
			continue
		}
		cp := *conn
		cp.Config = cloneMap(conn.Config)
		cp.Metadata = cloneMap(conn.Metadata)
		cp.SourceIDs = append([]string(nil), conn.SourceIDs...)
		out = append(out, &cp)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})
	return out, nil
}

func (s *InMemoryStore) Delete(ctx context.Context, id string) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.connections[id]; !ok {
		return fmt.Errorf("connection not found")
	}
	delete(s.connections, id)
	return nil
}

func normalizeProvider(provider string) string {
	provider = strings.ToLower(strings.TrimSpace(provider))
	switch provider {
	case "google_drive", "drive":
		return ProviderGDrive
	case "web", "crawler":
		return ProviderWebCrawler
	default:
		return provider
	}
}

func initialStatus(provider string, config map[string]interface{}) string {
	switch provider {
	case ProviderNotion, ProviderGDrive, ProviderSlack:
		if stringValue(config, "access_token", "token") == "" {
			return StatusOAuthRequired
		}
	case ProviderS3:
		if stringValue(config, "bucket") == "" {
			return StatusNotConfigured
		}
	case ProviderGitHub:
		if stringValue(config, "owner") == "" || stringValue(config, "repo") == "" {
			return StatusNotConfigured
		}
	case ProviderWebCrawler:
		if stringValue(config, "start_url", "url") == "" {
			return StatusNotConfigured
		}
	}
	return StatusActive
}

func statusForMissingConfig(provider string, config map[string]interface{}) string {
	status := initialStatus(provider, config)
	if status == StatusActive {
		return StatusError
	}
	return status
}

func stringValue(m map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if v, ok := m[key]; ok {
			switch typed := v.(type) {
			case string:
				return strings.TrimSpace(typed)
			default:
				return strings.TrimSpace(fmt.Sprint(typed))
			}
		}
	}
	return ""
}

func limitOrDefault(limit, fallback int) int {
	if limit <= 0 {
		return fallback
	}
	if limit > 100 {
		return 100
	}
	return limit
}

func documentType(doc SeedDocument) string {
	if strings.TrimSpace(doc.Type) != "" {
		return doc.Type
	}
	if doc.URL != "" && strings.TrimSpace(doc.Content) == "" {
		return "url"
	}
	return "text"
}

func hostTitle(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Host == "" {
		return rawURL
	}
	return parsed.Host
}

func cloneMap(in map[string]interface{}) map[string]interface{} {
	out := map[string]interface{}{}
	for k, v := range in {
		out[k] = v
	}
	return out
}

func mergeMaps(maps ...map[string]interface{}) map[string]interface{} {
	out := map[string]interface{}{}
	for _, m := range maps {
		for k, v := range m {
			out[k] = v
		}
	}
	return out
}

func redactMap(in map[string]interface{}) map[string]interface{} {
	out := map[string]interface{}{}
	for k, v := range in {
		lower := strings.ToLower(k)
		if strings.Contains(lower, "token") || strings.Contains(lower, "secret") || strings.Contains(lower, "password") || strings.Contains(lower, "key") {
			out[k] = "***redacted***"
			continue
		}
		out[k] = v
	}
	return out
}
