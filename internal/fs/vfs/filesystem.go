package vfs

import (
	"context"
	"fmt"
	"path"
	"strings"
	"sync"
	"time"

	"agent-memory/internal/memory/types"
	"agent-memory/internal/storage"
)

// VirtualFS presents agent memory as a filesystem tree:
//
//	/
//	├── memories/          tenant-scoped .md files
//	├── skills/            skill definitions as .md
//	├── sessions/          session ids as dirs
//	├── entities/          entity ids as .json
//	├── search/            write query.txt → read results.md
//	└── archive/           optional S3/GCS blob keys (when blob store set)
type VirtualFS struct {
	svc        ServiceInterface
	inodeMgr   *InodeManager
	cache      CacheInterface
	rootDir    *Directory
	mountPoint string
	tenantID   string
	blob       storage.BlobStore // optional archive backend
	blobPrefix string
	mu         sync.RWMutex
	stats      FSStats
	startTime  time.Time
}

// CacheInterface defines the cache operations
type CacheInterface interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{})
	Delete(key string)
	Clear()
}

// FSStats tracks filesystem statistics
type FSStats struct {
	TotalMemories int64
	TotalEntities int64
	TotalSkills   int64
	TotalSessions int64
	CacheHits     int64
	CacheMisses   int64
	ReadOps       int64
	WriteOps      int64
	DeleteOps     int64
	Uptime        time.Duration
}

type memoryCache struct {
	mu    sync.RWMutex
	items map[string]interface{}
}

func newMemoryCache(maxSize int) *memoryCache {
	return &memoryCache{items: make(map[string]interface{}, maxSize)}
}

func (c *memoryCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.items[key]
	return v, ok
}

func (c *memoryCache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = value
}

func (c *memoryCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

func (c *memoryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]interface{})
}

// NewVirtualFS creates a new virtual filesystem over a memory backend.
func NewVirtualFS(svc ServiceInterface, mountPoint string) *VirtualFS {
	return NewVirtualFSWithOptions(svc, mountPoint, "", nil, "")
}

// NewVirtualFSWithOptions configures tenant + optional blob archive.
func NewVirtualFSWithOptions(svc ServiceInterface, mountPoint, tenantID string, blob storage.BlobStore, blobPrefix string) *VirtualFS {
	if tenantID == "" {
		tenantID = "default"
	}
	if blobPrefix == "" {
		blobPrefix = "agentfs/" + tenantID + "/"
	}
	fs := &VirtualFS{
		svc:        svc,
		inodeMgr:   NewInodeManager(100000),
		cache:      newMemoryCache(10000),
		mountPoint: mountPoint,
		tenantID:   tenantID,
		blob:       blob,
		blobPrefix: blobPrefix,
		rootDir:    newRootDirectory(),
		startTime:  time.Now(),
	}
	fs.buildDirectoryTree()
	return fs
}

func (fs *VirtualFS) buildDirectoryTree() {
	fs.addDir("/", "memories", true)
	fs.addDir("/", "skills", true)
	fs.addDir("/", "sessions", true)
	fs.addDir("/", "entities", true)
	fs.addDir("/", "search", true)
	fs.addDir("/", "archive", true)
	// Convenience: drop new.md under memories to create a memory
	fs.inodeMgr.Allocate("/memories/new.md", "new.md", false, "new", "")
}

func (fs *VirtualFS) addDir(parentPath, name string, createInode bool) *Directory {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	parentDir := fs.rootDir
	if parentPath != "/" {
		if _, parentInode, ok := fs.inodeMgr.GetByPath(parentPath); ok && parentInode != nil {
			_ = parentInode
		}
	}
	if createInode {
		full := path.Join(parentPath, name)
		if parentPath == "/" {
			full = "/" + name
		}
		fs.inodeMgr.Allocate(full, name, true, "", "")
	}
	if parentDir.Children == nil {
		parentDir.Children = make(map[string]*DirEntry)
	}
	if _, ok := parentDir.Children[name]; !ok {
		parentDir.Children[name] = &DirEntry{Name: name, IsDir: true}
	}
	return parentDir
}

// ReadDir lists directory entries at a path
func (fs *VirtualFS) ReadDir(ctx context.Context, dirPath string) ([]DirEntry, error) {
	fs.mu.Lock()
	fs.stats.ReadOps++
	fs.mu.Unlock()

	dirPath = cleanPath(dirPath)

	if cached, ok := fs.cache.Get("dir:" + dirPath); ok {
		fs.mu.Lock()
		fs.stats.CacheHits++
		fs.mu.Unlock()
		if entries, ok := cached.([]DirEntry); ok {
			return entries, nil
		}
	}
	fs.mu.Lock()
	fs.stats.CacheMisses++
	fs.mu.Unlock()

	var (
		entries []DirEntry
		err     error
	)
	switch {
	case dirPath == "/":
		entries = []DirEntry{
			{Name: "memories", IsDir: true},
			{Name: "skills", IsDir: true},
			{Name: "sessions", IsDir: true},
			{Name: "entities", IsDir: true},
			{Name: "search", IsDir: true},
			{Name: "archive", IsDir: true},
		}
	case dirPath == "/memories":
		entries, err = fs.readMemoriesDir(ctx)
	case dirPath == "/skills":
		entries, err = fs.readSkillsDir(ctx)
	case dirPath == "/sessions":
		entries, err = fs.readSessionsDir(ctx)
	case dirPath == "/entities":
		entries, err = fs.readEntitiesDir(ctx)
	case dirPath == "/search":
		entries = []DirEntry{
			{Name: "query.txt", IsDir: false},
			{Name: "results.md", IsDir: false},
		}
	case dirPath == "/archive":
		entries, err = fs.readArchiveDir(ctx)
	default:
		return nil, fmt.Errorf("directory not found: %s", dirPath)
	}
	if err != nil {
		return nil, err
	}
	fs.cache.Set("dir:"+dirPath, entries)
	return entries, nil
}

func (fs *VirtualFS) readMemoriesDir(ctx context.Context) ([]DirEntry, error) {
	mems, err := fs.svc.GetMemoriesByTenant(ctx, fs.tenantID, 1000)
	if err != nil {
		// Fall back to search
		results, sErr := fs.svc.SearchMemories(ctx, &types.SearchRequest{
			Query:    "*",
			Limit:    100,
			TenantID: fs.tenantID,
		})
		if sErr != nil {
			return []DirEntry{{Name: "new.md", IsDir: false}}, nil
		}
		entries := []DirEntry{{Name: "new.md", IsDir: false}}
		for _, r := range results {
			id := r.MemoryID
			if id == "" {
				id = r.Entity.ID
			}
			if id == "" {
				continue
			}
			name := id + ".md"
			fp := "/memories/" + name
			inodeID, _ := fs.inodeMgr.Allocate(fp, name, false, id, "")
			entries = append(entries, DirEntry{Name: name, Inode: inodeID, IsDir: false})
		}
		return entries, nil
	}
	entries := []DirEntry{{Name: "new.md", IsDir: false}}
	for _, m := range mems {
		if m == nil || m.ID == "" {
			continue
		}
		name := m.ID + ".md"
		fp := "/memories/" + name
		inodeID, inode := fs.inodeMgr.Allocate(fp, name, false, m.ID, m.EntityID)
		if inode != nil {
			inode.Size = uint64(len(m.Content))
			inode.ModTime = m.UpdatedAt
		}
		entries = append(entries, DirEntry{Name: name, Inode: inodeID, IsDir: false})
	}
	return entries, nil
}

func (fs *VirtualFS) readSkillsDir(ctx context.Context) ([]DirEntry, error) {
	skills, err := fs.svc.ListSkills(ctx, fs.tenantID, "", 200, 0)
	if err != nil {
		return []DirEntry{}, nil
	}
	entries := make([]DirEntry, 0, len(skills))
	for _, sk := range skills {
		if sk == nil {
			continue
		}
		name := sanitizeName(sk.Name)
		if name == "" {
			name = sk.ID
		}
		name += ".md"
		fp := "/skills/" + name
		id, _ := fs.inodeMgr.Allocate(fp, name, false, sk.ID, "")
		entries = append(entries, DirEntry{Name: name, Inode: id, IsDir: false})
	}
	return entries, nil
}

func (fs *VirtualFS) readSessionsDir(ctx context.Context) ([]DirEntry, error) {
	sessions, err := fs.svc.ListSessions(ctx, "")
	if err != nil {
		return []DirEntry{}, nil
	}
	entries := make([]DirEntry, 0, len(sessions))
	for _, s := range sessions {
		if s == nil || s.ID == "" {
			continue
		}
		name := s.ID
		fp := "/sessions/" + name
		id, _ := fs.inodeMgr.Allocate(fp, name, true, s.ID, "")
		entries = append(entries, DirEntry{Name: name, Inode: id, IsDir: true})
	}
	return entries, nil
}

func (fs *VirtualFS) readEntitiesDir(ctx context.Context) ([]DirEntry, error) {
	// Entities listed via empty search of memories metadata is limited;
	// return placeholder until entity list API is always available.
	return []DirEntry{}, nil
}

func (fs *VirtualFS) readArchiveDir(ctx context.Context) ([]DirEntry, error) {
	if fs.blob == nil {
		return []DirEntry{{Name: "README.txt", IsDir: false}}, nil
	}
	keys, err := fs.blob.List(ctx, fs.blobPrefix)
	if err != nil {
		return []DirEntry{}, nil
	}
	entries := make([]DirEntry, 0, len(keys))
	for _, k := range keys {
		name := strings.TrimPrefix(k, fs.blobPrefix)
		if name == "" {
			continue
		}
		entries = append(entries, DirEntry{Name: name, IsDir: false})
	}
	return entries, nil
}

// ReadFile reads a file's content at a path
func (fs *VirtualFS) ReadFile(ctx context.Context, filePath string) ([]byte, error) {
	fs.mu.Lock()
	fs.stats.ReadOps++
	fs.mu.Unlock()

	filePath = cleanPath(filePath)

	if cached, ok := fs.cache.Get("file:" + filePath); ok {
		fs.mu.Lock()
		fs.stats.CacheHits++
		fs.mu.Unlock()
		if content, ok := cached.([]byte); ok {
			return content, nil
		}
	}
	fs.mu.Lock()
	fs.stats.CacheMisses++
	fs.mu.Unlock()

	switch {
	case filePath == "/memories/new.md":
		return []byte("---\n# New memory\n# Write content and save; AgentFS creates a memory.\n---\n\n"), nil
	case strings.HasPrefix(filePath, "/memories/") && strings.HasSuffix(filePath, ".md"):
		return fs.readMemoryFile(ctx, filePath)
	case strings.HasPrefix(filePath, "/skills/") && strings.HasSuffix(filePath, ".md"):
		return fs.readSkillFile(ctx, filePath)
	case filePath == "/search/query.txt":
		if v, ok := fs.cache.Get("search:query"); ok {
			if b, ok := v.([]byte); ok {
				return b, nil
			}
		}
		return []byte(""), nil
	case filePath == "/search/results.md":
		if v, ok := fs.cache.Get("search:results"); ok {
			if b, ok := v.([]byte); ok {
				return b, nil
			}
		}
		return []byte("# Search results\n\nWrite a query to /search/query.txt and save.\n"), nil
	case strings.HasPrefix(filePath, "/archive/"):
		return fs.readArchiveFile(ctx, filePath)
	case filePath == "/archive/README.txt":
		return []byte("Blob archive backend (S3/GCS). Set STORAGE_PROVIDER=s3|gcs.\n"), nil
	default:
		return nil, fmt.Errorf("file not found: %s", filePath)
	}
}

func (fs *VirtualFS) readMemoryFile(ctx context.Context, filePath string) ([]byte, error) {
	filename := path.Base(filePath)
	memoryID := strings.TrimSuffix(filename, ".md")
	if memoryID == "" || memoryID == "new" {
		return []byte(""), nil
	}
	mem, err := fs.svc.GetMemory(ctx, memoryID)
	if err != nil {
		return nil, fmt.Errorf("get memory: %w", err)
	}
	content := formatMemoryAsMarkdown(mem)
	fs.cache.Set("file:"+filePath, []byte(content))
	if id, _, ok := fs.inodeMgr.GetByPath(filePath); ok {
		fs.inodeMgr.UpdateSize(id, uint64(len(content)))
	}
	return []byte(content), nil
}

func formatMemoryAsMarkdown(mem *types.Memory) string {
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("id: %s\n", mem.ID))
	sb.WriteString(fmt.Sprintf("tenant_id: %s\n", mem.TenantID))
	sb.WriteString(fmt.Sprintf("user_id: %s\n", mem.UserID))
	sb.WriteString(fmt.Sprintf("created_at: %s\n", mem.CreatedAt.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("importance: %s\n", mem.Importance))
	if len(mem.Tags) > 0 {
		sb.WriteString(fmt.Sprintf("tags: %s\n", strings.Join(mem.Tags, ", ")))
	}
	sb.WriteString("---\n\n")
	sb.WriteString(mem.Content)
	sb.WriteString("\n")
	return sb.String()
}

func (fs *VirtualFS) readSkillFile(ctx context.Context, filePath string) ([]byte, error) {
	// Best-effort: list and match by name
	skills, err := fs.svc.ListSkills(ctx, fs.tenantID, "", 200, 0)
	if err != nil {
		return nil, err
	}
	base := strings.TrimSuffix(path.Base(filePath), ".md")
	for _, sk := range skills {
		if sk == nil {
			continue
		}
		if sanitizeName(sk.Name) == base || sk.ID == base {
			var sb strings.Builder
			sb.WriteString("---\n")
			sb.WriteString(fmt.Sprintf("id: %s\n", sk.ID))
			sb.WriteString(fmt.Sprintf("name: %s\n", sk.Name))
			sb.WriteString(fmt.Sprintf("domain: %s\n", sk.Domain))
			sb.WriteString(fmt.Sprintf("trigger: %s\n", sk.Trigger))
			sb.WriteString("---\n\n")
			sb.WriteString("## Trigger\n\n")
			sb.WriteString(sk.Trigger)
			sb.WriteString("\n\n## Action\n\n")
			sb.WriteString(sk.Action)
			sb.WriteString("\n")
			return []byte(sb.String()), nil
		}
	}
	return nil, fmt.Errorf("skill not found: %s", base)
}

func (fs *VirtualFS) readArchiveFile(ctx context.Context, filePath string) ([]byte, error) {
	if fs.blob == nil {
		return nil, fmt.Errorf("archive backend not configured")
	}
	name := strings.TrimPrefix(filePath, "/archive/")
	return fs.blob.Download(ctx, fs.blobPrefix+name)
}

// WriteFile writes content to a file (creates or updates memory / search / archive).
func (fs *VirtualFS) WriteFile(ctx context.Context, filePath string, content []byte) error {
	fs.mu.Lock()
	fs.stats.WriteOps++
	fs.mu.Unlock()

	filePath = cleanPath(filePath)

	switch {
	case filePath == "/search/query.txt":
		fs.cache.Set("search:query", content)
		q := strings.TrimSpace(string(content))
		if q == "" {
			return nil
		}
		results, err := fs.svc.SearchMemories(ctx, &types.SearchRequest{
			Query:    q,
			Limit:    20,
			TenantID: fs.tenantID,
		})
		if err != nil {
			fs.cache.Set("search:results", []byte("# Search error\n\n"+err.Error()+"\n"))
			return nil
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("# Search results for %q\n\n", q))
		for i, r := range results {
			id := r.MemoryID
			text := r.Text
			if text == "" && r.Metadata != nil {
				text = r.Metadata.Content
			}
			sb.WriteString(fmt.Sprintf("## %d. %s (score %.3f)\n\n%s\n\n", i+1, id, r.Score, text))
		}
		fs.cache.Set("search:results", []byte(sb.String()))
		return nil

	case strings.HasPrefix(filePath, "/archive/"):
		if fs.blob == nil {
			return fmt.Errorf("archive backend not configured (set STORAGE_PROVIDER=s3|gcs)")
		}
		name := strings.TrimPrefix(filePath, "/archive/")
		return fs.blob.Upload(ctx, fs.blobPrefix+name, content)

	case strings.HasPrefix(filePath, "/memories/") && strings.HasSuffix(filePath, ".md"):
		return fs.writeMemoryFile(ctx, filePath, content)

	default:
		return fmt.Errorf("write not supported for %s", filePath)
	}
}

func (fs *VirtualFS) writeMemoryFile(ctx context.Context, filePath string, content []byte) error {
	filename := path.Base(filePath)
	memoryID := strings.TrimSuffix(filename, ".md")
	body, meta := parseMarkdownMemory(string(content))

	if memoryID == "new" || memoryID == "" {
		mem := &types.Memory{
			Content:  body,
			TenantID: fs.tenantID,
			Type:     types.MemoryTypeUser,
		}
		if v, ok := meta["user_id"]; ok {
			mem.UserID = v
		}
		created, err := fs.svc.CreateMemory(ctx, mem)
		if err != nil {
			return fmt.Errorf("create memory: %w", err)
		}
		// Also stash full markdown in archive if configured
		if fs.blob != nil {
			_ = fs.blob.Upload(ctx, fs.blobPrefix+"memories/"+created.ID+".md", content)
		}
		fs.cache.Delete("dir:/memories")
		return nil
	}

	if err := fs.svc.UpdateMemory(ctx, memoryID, body, nil); err != nil {
		return fmt.Errorf("update memory: %w", err)
	}
	if fs.blob != nil {
		_ = fs.blob.Upload(ctx, fs.blobPrefix+"memories/"+memoryID+".md", content)
	}
	fs.cache.Delete("file:" + filePath)
	fs.cache.Delete("dir:/memories")
	return nil
}

// DeleteFile deletes a file (memory or archive object).
func (fs *VirtualFS) DeleteFile(ctx context.Context, filePath string) error {
	fs.mu.Lock()
	fs.stats.DeleteOps++
	fs.mu.Unlock()

	filePath = cleanPath(filePath)

	if strings.HasPrefix(filePath, "/archive/") {
		if fs.blob == nil {
			return fmt.Errorf("archive backend not configured")
		}
		name := strings.TrimPrefix(filePath, "/archive/")
		return fs.blob.Delete(ctx, fs.blobPrefix+name)
	}

	if !strings.HasPrefix(filePath, "/memories/") || !strings.HasSuffix(filePath, ".md") {
		return fmt.Errorf("delete only supported for memory and archive files")
	}
	memoryID := strings.TrimSuffix(path.Base(filePath), ".md")
	if memoryID == "new" {
		return nil
	}
	if err := fs.svc.DeleteMemory(ctx, memoryID); err != nil {
		return err
	}
	fs.inodeMgr.DeleteByPath(filePath)
	fs.cache.Delete("file:" + filePath)
	fs.cache.Delete("dir:/memories")
	return nil
}

// GetAttr returns file attributes for a path
func (fs *VirtualFS) GetAttr(filePath string) (*FileAttr, error) {
	filePath = cleanPath(filePath)
	if id, inode, ok := fs.inodeMgr.GetByPath(filePath); ok {
		_ = id
		return &FileAttr{
			Ino:    inode.ID,
			Size:   inode.Size,
			Blocks: (inode.Size + 511) / 512,
			Atime:  uint64(inode.ModTime.Unix()),
			Mtime:  uint64(inode.ModTime.Unix()),
			Ctime:  uint64(inode.ModTime.Unix()),
			Mode:   inode.Mode,
			Nlink:  1,
		}, nil
	}
	// Synthetic dirs
	switch filePath {
	case "/", "/memories", "/skills", "/sessions", "/entities", "/search", "/archive":
		return &FileAttr{Mode: 0o755 | 0o40000, Nlink: 2}, nil
	}
	return nil, fmt.Errorf("path not found: %s", filePath)
}

// GetStats returns filesystem statistics
func (fs *VirtualFS) GetStats() FSStats {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	stats := fs.stats
	stats.Uptime = time.Since(fs.startTime)
	stats.TotalMemories = int64(fs.inodeMgr.Count())
	return stats
}

func (fs *VirtualFS) GetInodeByPath(p string) (uint64, *Inode, bool) {
	return fs.inodeMgr.GetByPath(cleanPath(p))
}

func (fs *VirtualFS) AddDir(parentPath, name string) {
	fs.addDir(parentPath, name, true)
}

// Status returns a human-readable status line.
func (fs *VirtualFS) Status() string {
	st := fs.GetStats()
	backend := "memory-api"
	if fs.blob != nil {
		backend += "+blob"
	}
	return fmt.Sprintf("agentfs tenant=%s backend=%s reads=%d writes=%d deletes=%d uptime=%s",
		fs.tenantID, backend, st.ReadOps, st.WriteOps, st.DeleteOps, st.Uptime.Round(time.Second))
}

func newRootDirectory() *Directory {
	return &Directory{
		Inode: &Inode{
			ID:    1,
			Mode:  0o755 | 0o40000,
			IsDir: true,
			Name:  "/",
			Path:  "/",
		},
		Children: map[string]*DirEntry{
			"memories": {Name: "memories", IsDir: true},
			"skills":   {Name: "skills", IsDir: true},
			"sessions": {Name: "sessions", IsDir: true},
			"entities": {Name: "entities", IsDir: true},
			"search":   {Name: "search", IsDir: true},
			"archive":  {Name: "archive", IsDir: true},
		},
	}
}

func cleanPath(p string) string {
	if p == "" {
		return "/"
	}
	p = path.Clean("/" + strings.TrimPrefix(p, "/"))
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return p
}

func sanitizeName(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, " ", "_")
	return s
}

// parseMarkdownMemory splits optional YAML-like front matter from body.
func parseMarkdownMemory(raw string) (body string, meta map[string]string) {
	meta = map[string]string{}
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, "---") {
		return raw, meta
	}
	rest := strings.TrimPrefix(raw, "---")
	parts := strings.SplitN(rest, "---", 2)
	if len(parts) < 2 {
		return raw, meta
	}
	for _, line := range strings.Split(parts[0], "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		kv := strings.SplitN(line, ":", 2)
		if len(kv) == 2 {
			meta[strings.ToLower(strings.TrimSpace(kv[0]))] = strings.TrimSpace(kv[1])
		}
	}
	return strings.TrimSpace(parts[1]), meta
}
