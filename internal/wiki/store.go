package wiki

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"agent-memory/internal/memory/types"
)

type Store interface {
	// Pages
	GetPage(ctx context.Context, id string) (*Page, error)
	SavePage(ctx context.Context, page *Page) error
	DeletePage(ctx context.Context, id string) error
	ListPages(ctx context.Context, limit, offset int) ([]*Page, int64, error)
	SearchPages(ctx context.Context, query string, limit int) ([]*Page, error)

	// Sources
	GetSource(ctx context.Context, id string) (*Source, error)
	SaveSource(ctx context.Context, source *Source) error
	DeleteSource(ctx context.Context, id string) error
	ListSources(ctx context.Context, limit, offset int) ([]*Source, int64, error)

	// Logs
	GetLogs(ctx context.Context, limit int) ([]LogEntry, error)
	AddLog(ctx context.Context, entry LogEntry) error

	// Memory integration
	CreateMemory(ctx context.Context, mem *types.Memory) error
	GetMemory(ctx context.Context, id string) (*types.Memory, error)
	DeleteMemory(ctx context.Context, id string) error

	// Indexing
	UpdateIndex(ctx context.Context) error
	GetIndex(ctx context.Context) (*Index, error)

	// Load all data on startup
	Load(ctx context.Context) error
}

type Index struct {
	GeneratedAt time.Time `json:"generated_at"`
	PageCount   int       `json:"page_count"`
	SourceCount int       `json:"source_count"`
}

type FilesystemStore struct {
	baseDir string
	mu      sync.RWMutex
	pages   map[string]*Page
	sources map[string]*Source
	logs    []LogEntry
	index   *Index
}

func NewFilesystemStore(baseDir string) *FilesystemStore {
	return &FilesystemStore{baseDir: baseDir}
}

func (f *FilesystemStore) path(name string) string {
	return filepath.Join(f.baseDir, name+".json")
}

func (f *FilesystemStore) Load(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Load pages
	if data, err := os.ReadFile(f.path("pages")); err == nil {
		if err := json.Unmarshal(data, &f.pages); err != nil {
			return fmt.Errorf("load pages: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("load pages: %w", err)
	}
	// Initialize pages map if not loaded
	if f.pages == nil {
		f.pages = make(map[string]*Page)
	}

	// Load sources
	if data, err := os.ReadFile(f.path("sources")); err == nil {
		if err := json.Unmarshal(data, &f.sources); err != nil {
			return fmt.Errorf("load sources: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("load sources: %w", err)
	}
	// Initialize sources map if not loaded
	if f.sources == nil {
		f.sources = make(map[string]*Source)
	}

	// Load logs
	if data, err := os.ReadFile(f.path("logs")); err == nil {
		if err := json.Unmarshal(data, &f.logs); err != nil {
			return fmt.Errorf("load logs: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("load logs: %w", err)
	}
	// Initialize logs if not loaded
	if f.logs == nil {
		f.logs = []LogEntry{}
	}

	// Load index
	if data, err := os.ReadFile(f.path("index")); err == nil {
		if err := json.Unmarshal(data, &f.index); err != nil {
			return fmt.Errorf("load index: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("load index: %w", err)
	}

	return nil
}

func (f *FilesystemStore) saveAll() error {
	// Ensure directory exists
	if err := os.MkdirAll(f.baseDir, 0755); err != nil {
		return fmt.Errorf("create wiki dir: %w", err)
	}

	// Save pages
	if data, err := json.MarshalIndent(f.pages, "", "  "); err == nil {
		if err := os.WriteFile(f.path("pages"), data, 0644); err != nil {
			return fmt.Errorf("save pages: %w", err)
		}
	}

	// Save sources
	if data, err := json.MarshalIndent(f.sources, "", "  "); err == nil {
		if err := os.WriteFile(f.path("sources"), data, 0644); err != nil {
			return fmt.Errorf("save sources: %w", err)
		}
	}

	// Save logs
	if data, err := json.MarshalIndent(f.logs, "", "  "); err == nil {
		if err := os.WriteFile(f.path("logs"), data, 0644); err != nil {
			return fmt.Errorf("save logs: %w", err)
		}
	}

	// Save index
	if f.index != nil {
		if data, err := json.MarshalIndent(f.index, "", "  "); err == nil {
			if err := os.WriteFile(f.path("index"), data, 0644); err != nil {
				return fmt.Errorf("save index: %w", err)
			}
		}
	}

	return nil
}

func (f *FilesystemStore) GetPage(ctx context.Context, id string) (*Page, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	page, ok := f.pages[id]
	if !ok {
		return nil, fmt.Errorf("page not found: %s", id)
	}
	return page, nil
}

func (f *FilesystemStore) SavePage(ctx context.Context, page *Page) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.pages[page.ID] = page
	return f.saveAll()
}

func (f *FilesystemStore) DeletePage(ctx context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.pages, id)
	return f.saveAll()
}

func (f *FilesystemStore) ListPages(ctx context.Context, limit, offset int) ([]*Page, int64, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	var result []*Page
	for _, p := range f.pages {
		result = append(result, p)
		if limit > 0 && len(result) >= limit {
			break
		}
	}
	count := int64(len(f.pages))
	return result, count, nil
}

func (f *FilesystemStore) SearchPages(ctx context.Context, query string, limit int) ([]*Page, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	var result []*Page
	ql := strings.ToLower(query)
	for _, p := range f.pages {
		if strings.Contains(strings.ToLower(p.Title), ql) || strings.Contains(strings.ToLower(p.Content), ql) {
			result = append(result, p)
			if len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (f *FilesystemStore) GetSource(ctx context.Context, id string) (*Source, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	src, ok := f.sources[id]
	if !ok {
		return nil, fmt.Errorf("source not found: %s", id)
	}
	return src, nil
}

func (f *FilesystemStore) SaveSource(ctx context.Context, source *Source) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sources[source.ID] = source
	return f.saveAll()
}

func (f *FilesystemStore) DeleteSource(ctx context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.sources, id)
	return f.saveAll()
}

func (f *FilesystemStore) ListSources(ctx context.Context, limit, offset int) ([]*Source, int64, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	var result []*Source
	for _, s := range f.sources {
		result = append(result, s)
		if limit > 0 && len(result) >= limit {
			break
		}
	}
	count := int64(len(f.sources))
	return result, count, nil
}

func (f *FilesystemStore) GetLogs(ctx context.Context, limit int) ([]LogEntry, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if limit <= 0 || limit > len(f.logs) {
		return f.logs, nil
	}
	return f.logs[len(f.logs)-limit:], nil
}

func (f *FilesystemStore) AddLog(ctx context.Context, entry LogEntry) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.logs = append(f.logs, entry)
	return f.saveAll()
}

func (f *FilesystemStore) CreateMemory(ctx context.Context, mem *types.Memory) error {
	// Could be implemented to persist memory metadata, but for now just return nil
	return nil
}

func (f *FilesystemStore) GetMemory(ctx context.Context, id string) (*types.Memory, error) {
	// Could load from memory service, but store doesn't handle this
	return nil, fmt.Errorf("memory not found")
}

func (f *FilesystemStore) DeleteMemory(ctx context.Context, id string) error {
	// Could delete memory, but not implemented
	return nil
}

func (f *FilesystemStore) UpdateIndex(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.index = &Index{
		GeneratedAt: time.Now(),
		PageCount:   len(f.pages),
		SourceCount: len(f.sources),
	}
	return f.saveAll()
}

func (f *FilesystemStore) GetIndex(ctx context.Context) (*Index, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.index == nil {
		// Build index inline while holding the lock (no re-entrancy needed)
		f.index = &Index{
			GeneratedAt: time.Now(),
			PageCount:   len(f.pages),
			SourceCount: len(f.sources),
		}
		// Best-effort save; ignore error since we can still return the index
		_ = f.saveAll()
	}
	return f.index, nil
}
