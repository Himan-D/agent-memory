package sources

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"agent-memory/internal/extractors"
	"agent-memory/internal/memory/types"
)

const (
	CategorySource      = "source"
	CategorySourceChunk = "source_chunk"
)

type MemoryService interface {
	CreateMemory(ctx context.Context, mem *types.Memory) (*types.Memory, error)
	GetMemory(ctx context.Context, id string) (*types.Memory, error)
	DeleteMemory(ctx context.Context, id string) error
	GetMemoriesByUser(ctx context.Context, userID string) ([]*types.Memory, error)
	GetMemoriesByOrg(ctx context.Context, orgID string) ([]*types.Memory, error)
	SearchMemories(ctx context.Context, req *types.SearchRequest) ([]types.MemoryResult, error)
}

type BlobStore interface {
	Put(ctx context.Context, key string, data []byte, contentType string) error
	Delete(ctx context.Context, key string) error
}

type Config struct {
	DataDir         string
	ChunkMaxBytes   int
	StorageProvider string
}

type Service struct {
	mem        MemoryService
	blobs      BlobStore
	extractors *extractors.Registry
	httpClient *http.Client
	cfg        Config
}

type Source struct {
	ID              string                 `json:"id"`
	Title           string                 `json:"title"`
	Type            string                 `json:"type"`
	Provider        string                 `json:"provider"`
	ExternalID      string                 `json:"external_id,omitempty"`
	URL             string                 `json:"url,omitempty"`
	R2Key           string                 `json:"r2_key,omitempty"`
	ContentHash     string                 `json:"content_hash"`
	MimeType        string                 `json:"mime_type,omitempty"`
	Bytes           int                    `json:"bytes,omitempty"`
	UserID          string                 `json:"user_id,omitempty"`
	OrgID           string                 `json:"org_id,omitempty"`
	AgentID         string                 `json:"agent_id,omitempty"`
	SourceMemoryID  string                 `json:"source_memory_id"`
	ChunkMemoryIDs  []string               `json:"chunk_memory_ids"`
	ChunksCreated   int                    `json:"chunks_created"`
	MemoriesCreated int                    `json:"memories_created"`
	Status          string                 `json:"status"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

type IngestRequest struct {
	Type       string                 `json:"type"`
	Content    string                 `json:"content,omitempty"`
	URL        string                 `json:"url,omitempty"`
	Title      string                 `json:"title,omitempty"`
	Provider   string                 `json:"provider,omitempty"`
	ExternalID string                 `json:"external_id,omitempty"`
	UserID     string                 `json:"user_id,omitempty"`
	OrgID      string                 `json:"org_id,omitempty"`
	AgentID    string                 `json:"agent_id,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

type UploadRequest struct {
	Filename    string
	ContentType string
	Reader      io.Reader
	Title       string
	UserID      string
	OrgID       string
	AgentID     string
	Metadata    map[string]interface{}
}

type IngestResult struct {
	Source          *Source  `json:"source"`
	SourceID        string   `json:"source_id"`
	Status          string   `json:"status"`
	ChunksCreated   int      `json:"chunks_created"`
	MemoriesCreated int      `json:"memories_created"`
	EntitiesCreated int      `json:"entities_created"`
	MemoryIDs       []string `json:"memory_ids"`
	R2Key           string   `json:"r2_key,omitempty"`
	MimeType        string   `json:"mime_type,omitempty"`
	Bytes           int      `json:"bytes,omitempty"`
}

func NewService(mem MemoryService, blobs BlobStore, cfg Config) *Service {
	if cfg.ChunkMaxBytes <= 0 {
		cfg.ChunkMaxBytes = 2048
	}
	if blobs == nil {
		blobs = NewFilesystemBlobStore(filepath.Join(cfg.DataDir, "sources"))
	}
	return &Service{
		mem:        mem,
		blobs:      blobs,
		extractors: extractors.NewRegistry(),
		httpClient: &http.Client{Timeout: 30 * time.Second},
		cfg:        cfg,
	}
}

func (s *Service) Ingest(ctx context.Context, req IngestRequest) (*IngestResult, error) {
	if strings.TrimSpace(req.Type) == "" {
		req.Type = "text"
	}
	provider := req.Provider
	if provider == "" {
		provider = req.Type
	}

	content := req.Content
	mimeType := "text/plain"
	title := req.Title
	sourceURL := req.URL

	if req.Type == "url" || req.Type == "web" {
		doc, err := s.fetchURL(ctx, req.URL)
		if err != nil {
			return nil, err
		}
		content = doc.Content
		mimeType = doc.MimeType
		if title == "" {
			title = doc.Title
		}
		provider = "web"
	}
	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("source content is required")
	}
	if title == "" {
		title = defaultTitle(provider, sourceURL, content)
	}

	return s.persistDocument(ctx, documentInput{
		Type:       req.Type,
		Provider:   provider,
		ExternalID: req.ExternalID,
		Title:      title,
		URL:        sourceURL,
		Content:    content,
		MimeType:   mimeType,
		UserID:     req.UserID,
		OrgID:      req.OrgID,
		AgentID:    req.AgentID,
		Metadata:   req.Metadata,
	})
}

func (s *Service) Upload(ctx context.Context, req UploadRequest) (*IngestResult, error) {
	if req.Reader == nil {
		return nil, fmt.Errorf("file is required")
	}
	data, err := io.ReadAll(req.Reader)
	if err != nil {
		return nil, fmt.Errorf("read upload: %w", err)
	}
	contentType := req.ContentType
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}
	doc, err := s.extractors.Extract(contentType, bytes.NewReader(data), req.Filename)
	if err != nil {
		if ext, ok := s.extractors.FindByFilename(strings.ToLower(req.Filename)); ok {
			doc, err = ext.Extract(bytes.NewReader(data), req.Filename)
		}
	}
	if err != nil {
		doc = &extractors.Document{
			Content:   fmt.Sprintf("[Unsupported attachment: %s, %d bytes]", req.Filename, len(data)),
			Title:     strings.TrimSuffix(filepath.Base(req.Filename), filepath.Ext(req.Filename)),
			MimeType:  contentType,
			Source:    req.Filename,
			Metadata:  map[string]string{"unsupported": "true"},
			PageCount: 1,
		}
	}

	sourceID := uuid.New().String()
	key := fmt.Sprintf("tenants/%s/sources/%s/%s", tenantKey(req.OrgID, req.UserID), sourceID, sanitizeFilename(req.Filename))
	if err := s.blobs.Put(ctx, key, data, contentType); err != nil {
		return nil, fmt.Errorf("store upload: %w", err)
	}

	title := req.Title
	if title == "" {
		title = doc.Title
	}
	if title == "" {
		title = req.Filename
	}
	meta := cloneMap(req.Metadata)
	meta["filename"] = req.Filename
	meta["pages"] = doc.PageCount
	for k, v := range doc.Metadata {
		meta[k] = v
	}
	return s.persistDocument(ctx, documentInput{
		ID:       sourceID,
		Type:     "file",
		Provider: "file",
		Title:    title,
		Content:  doc.Content,
		MimeType: doc.MimeType,
		Bytes:    len(data),
		R2Key:    key,
		UserID:   req.UserID,
		OrgID:    req.OrgID,
		AgentID:  req.AgentID,
		Metadata: meta,
	})
}

func (s *Service) List(ctx context.Context, userID, orgID string, limit, offset int) ([]*Source, int, error) {
	memories, err := s.scopeMemories(ctx, userID, orgID)
	if err != nil {
		return nil, 0, err
	}
	sources := make([]*Source, 0)
	for _, mem := range memories {
		if mem.Category != CategorySource {
			continue
		}
		source := sourceFromMemory(mem)
		if source != nil {
			sources = append(sources, source)
		}
	}
	sort.Slice(sources, func(i, j int) bool {
		return sources[i].CreatedAt.After(sources[j].CreatedAt)
	})
	total := len(sources)
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 50
	}
	if offset > len(sources) {
		return []*Source{}, total, nil
	}
	end := offset + limit
	if end > len(sources) {
		end = len(sources)
	}
	return sources[offset:end], total, nil
}

func (s *Service) Get(ctx context.Context, sourceID string) (*Source, error) {
	mem, err := s.mem.GetMemory(ctx, sourceID)
	if err != nil {
		return nil, err
	}
	source := sourceFromMemory(mem)
	if source == nil {
		return nil, fmt.Errorf("source not found: %s", sourceID)
	}
	return source, nil
}

func (s *Service) Delete(ctx context.Context, sourceID string) error {
	source, err := s.Get(ctx, sourceID)
	if err != nil {
		return err
	}
	for _, id := range source.ChunkMemoryIDs {
		if err := s.mem.DeleteMemory(ctx, id); err != nil {
			return err
		}
	}
	if source.R2Key != "" {
		_ = s.blobs.Delete(ctx, source.R2Key)
	}
	return s.mem.DeleteMemory(ctx, sourceID)
}

type documentInput struct {
	ID         string
	Type       string
	Provider   string
	ExternalID string
	Title      string
	URL        string
	Content    string
	MimeType   string
	Bytes      int
	R2Key      string
	UserID     string
	OrgID      string
	AgentID    string
	Metadata   map[string]interface{}
}

func (s *Service) persistDocument(ctx context.Context, input documentInput) (*IngestResult, error) {
	sourceID := input.ID
	if sourceID == "" {
		sourceID = uuid.New().String()
	}
	contentHash := hashContent(input.Content)
	now := time.Now()
	chunks := chunkText(input.Content, s.cfg.ChunkMaxBytes)
	if len(chunks) == 0 {
		chunks = []string{input.Content}
	}

	source := &Source{
		ID:          sourceID,
		Title:       input.Title,
		Type:        input.Type,
		Provider:    input.Provider,
		ExternalID:  input.ExternalID,
		URL:         input.URL,
		R2Key:       input.R2Key,
		ContentHash: contentHash,
		MimeType:    input.MimeType,
		Bytes:       input.Bytes,
		UserID:      input.UserID,
		OrgID:       input.OrgID,
		AgentID:     input.AgentID,
		Status:      "active",
		Metadata:    cloneMap(input.Metadata),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	chunkIDs := make([]string, 0, len(chunks))
	for i, chunk := range chunks {
		chunkID := uuid.New().String()
		chunkMeta := sourceMetadata(source)
		chunkMeta["chunk_index"] = i
		chunkMeta["chunk_count"] = len(chunks)
		chunkMeta["content_hash"] = hashContent(chunk)
		mem := &types.Memory{
			ID:         chunkID,
			Content:    chunk,
			Type:       types.MemoryTypeUser,
			UserID:     input.UserID,
			OrgID:      input.OrgID,
			AgentID:    input.AgentID,
			Category:   CategorySourceChunk,
			Tags:       []string{"source", input.Provider},
			Importance: types.ImportanceMedium,
			Status:     types.MemoryStatusActive,
			Metadata:   chunkMeta,
			SourceType: types.SourceExternal,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		created, err := s.mem.CreateMemory(ctx, mem)
		if err != nil {
			return nil, fmt.Errorf("create chunk memory: %w", err)
		}
		chunkIDs = append(chunkIDs, created.ID)
	}
	source.ChunkMemoryIDs = chunkIDs
	source.ChunksCreated = len(chunks)
	source.MemoriesCreated = len(chunks) + 1

	sourceMeta := sourceMetadata(source)
	sourceMeta["chunk_memory_ids"] = chunkIDs
	sourceMem := &types.Memory{
		ID:         sourceID,
		Content:    fmt.Sprintf("[Source] %s", input.Title),
		Type:       types.MemoryTypeUser,
		UserID:     input.UserID,
		OrgID:      input.OrgID,
		AgentID:    input.AgentID,
		Category:   CategorySource,
		Tags:       []string{"source", input.Provider},
		Importance: types.ImportanceHigh,
		Status:     types.MemoryStatusActive,
		Metadata:   sourceMeta,
		SourceType: types.SourceExternal,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	createdSource, err := s.mem.CreateMemory(ctx, sourceMem)
	if err != nil {
		return nil, fmt.Errorf("create source memory: %w", err)
	}
	source.SourceMemoryID = createdSource.ID

	return &IngestResult{
		Source:          source,
		SourceID:        source.ID,
		Status:          source.Status,
		ChunksCreated:   source.ChunksCreated,
		MemoriesCreated: source.MemoriesCreated,
		EntitiesCreated: 0,
		MemoryIDs:       append([]string{source.ID}, chunkIDs...),
		R2Key:           source.R2Key,
		MimeType:        source.MimeType,
		Bytes:           source.Bytes,
	}, nil
}

func (s *Service) fetchURL(ctx context.Context, rawURL string) (*extractors.Document, error) {
	if rawURL == "" {
		return nil, fmt.Errorf("url is required")
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("invalid url: %s", rawURL)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("fetch url: HTTP %d", resp.StatusCode)
	}
	contentType := resp.Header.Get("Content-Type")
	if idx := strings.Index(contentType, ";"); idx >= 0 {
		contentType = strings.TrimSpace(contentType[:idx])
	}
	if contentType == "" {
		contentType = "text/html"
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read url body: %w", err)
	}
	doc, err := s.extractors.Extract(contentType, bytes.NewReader(body), parsed.Host)
	if err != nil {
		doc = &extractors.Document{
			Content:  string(body),
			Title:    parsed.Host,
			MimeType: contentType,
			Source:   rawURL,
		}
	}
	if doc.Title == "" {
		doc.Title = parsed.Host
	}
	doc.Source = rawURL
	return doc, nil
}

func (s *Service) scopeMemories(ctx context.Context, userID, orgID string) ([]*types.Memory, error) {
	if orgID != "" {
		return s.mem.GetMemoriesByOrg(ctx, orgID)
	}
	if userID != "" {
		return s.mem.GetMemoriesByUser(ctx, userID)
	}
	results, err := s.mem.SearchMemories(ctx, &types.SearchRequest{
		Query:     "source",
		Limit:     1000,
		Threshold: 0,
		Mode:      "hybrid",
	})
	if err != nil {
		return nil, err
	}
	memories := make([]*types.Memory, 0, len(results))
	for _, result := range results {
		if result.Metadata != nil {
			memories = append(memories, result.Metadata)
		}
	}
	return memories, nil
}

func sourceMetadata(source *Source) map[string]interface{} {
	meta := cloneMap(source.Metadata)
	meta["source_id"] = source.ID
	meta["source_type"] = source.Type
	meta["provider"] = source.Provider
	meta["external_id"] = source.ExternalID
	meta["title"] = source.Title
	meta["url"] = source.URL
	meta["r2_key"] = source.R2Key
	meta["content_hash"] = source.ContentHash
	meta["mime_type"] = source.MimeType
	meta["bytes"] = source.Bytes
	meta["source_memory_id"] = source.SourceMemoryID
	meta["chunks_created"] = source.ChunksCreated
	meta["status"] = source.Status
	return meta
}

func sourceFromMemory(mem *types.Memory) *Source {
	if mem == nil || mem.Category != CategorySource || mem.Metadata == nil {
		return nil
	}
	meta := mem.Metadata
	source := &Source{
		ID:             stringVal(meta["source_id"], mem.ID),
		Title:          stringVal(meta["title"], mem.Content),
		Type:           stringVal(meta["source_type"], ""),
		Provider:       stringVal(meta["provider"], ""),
		ExternalID:     stringVal(meta["external_id"], ""),
		URL:            stringVal(meta["url"], ""),
		R2Key:          stringVal(meta["r2_key"], ""),
		ContentHash:    stringVal(meta["content_hash"], ""),
		MimeType:       stringVal(meta["mime_type"], ""),
		Bytes:          intVal(meta["bytes"]),
		UserID:         mem.UserID,
		OrgID:          mem.OrgID,
		AgentID:        mem.AgentID,
		SourceMemoryID: mem.ID,
		ChunksCreated:  intVal(meta["chunks_created"]),
		Status:         stringVal(meta["status"], string(mem.Status)),
		Metadata:       meta,
		CreatedAt:      mem.CreatedAt,
		UpdatedAt:      mem.UpdatedAt,
	}
	source.ChunkMemoryIDs = stringSlice(meta["chunk_memory_ids"])
	source.MemoriesCreated = len(source.ChunkMemoryIDs) + 1
	return source
}

func chunkText(text string, maxBytes int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	paragraphs := strings.Split(text, "\n\n")
	chunks := make([]string, 0)
	var current strings.Builder
	flush := func() {
		if strings.TrimSpace(current.String()) != "" {
			chunks = append(chunks, strings.TrimSpace(current.String()))
			current.Reset()
		}
	}
	for _, paragraph := range paragraphs {
		paragraph = strings.TrimSpace(paragraph)
		if paragraph == "" {
			continue
		}
		if current.Len()+len(paragraph)+2 > maxBytes {
			flush()
		}
		if len(paragraph) > maxBytes {
			words := strings.Fields(paragraph)
			for _, word := range words {
				if current.Len()+len(word)+1 > maxBytes {
					flush()
				}
				if current.Len() > 0 {
					current.WriteByte(' ')
				}
				current.WriteString(word)
			}
			continue
		}
		if current.Len() > 0 {
			current.WriteString("\n\n")
		}
		current.WriteString(paragraph)
	}
	flush()
	return chunks
}

func defaultTitle(provider, sourceURL, content string) string {
	if sourceURL != "" {
		if parsed, err := url.Parse(sourceURL); err == nil && parsed.Host != "" {
			return parsed.Host
		}
	}
	fields := strings.Fields(content)
	if len(fields) > 8 {
		fields = fields[:8]
	}
	if len(fields) > 0 {
		return strings.Join(fields, " ")
	}
	return strings.Title(provider) + " Source"
}

func hashContent(content string) string {
	sum := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", sum)[:16]
}

func tenantKey(orgID, userID string) string {
	if orgID != "" {
		return sanitizeFilename(orgID)
	}
	if userID != "" {
		return sanitizeFilename(userID)
	}
	return "default"
}

var unsafeFilename = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func sanitizeFilename(name string) string {
	name = strings.TrimSpace(filepath.Base(name))
	if name == "" || name == "." || name == "/" {
		name = "upload"
	}
	return unsafeFilename.ReplaceAllString(name, "_")
}

func cloneMap(in map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{})
	for k, v := range in {
		out[k] = v
	}
	return out
}

func stringVal(v interface{}, fallback string) string {
	if s, ok := v.(string); ok && s != "" {
		return s
	}
	return fallback
}

func intVal(v interface{}) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	case float32:
		return int(n)
	default:
		return 0
	}
}

func stringSlice(v interface{}) []string {
	switch x := v.(type) {
	case []string:
		return x
	case []interface{}:
		out := make([]string, 0, len(x))
		for _, item := range x {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

type FilesystemBlobStore struct {
	baseDir string
}

func NewFilesystemBlobStore(baseDir string) *FilesystemBlobStore {
	return &FilesystemBlobStore{baseDir: baseDir}
}

func (f *FilesystemBlobStore) Put(ctx context.Context, key string, data []byte, contentType string) error {
	path := filepath.Join(f.baseDir, filepath.Clean(key))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (f *FilesystemBlobStore) Delete(ctx context.Context, key string) error {
	path := filepath.Join(f.baseDir, filepath.Clean(key))
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
