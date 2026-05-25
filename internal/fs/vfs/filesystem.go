package vfs

import (
	"context"
	"fmt"
	"path"
	"strings"
	"sync"
	"time"

	"agent-memory/internal/memory/types"
)

// VirtualFS implements the main filesystem logic
// Follows the pattern of memory.Service but for filesystem operations
type VirtualFS struct {
	svc        ServiceInterface
	inodeMgr   *InodeManager
	cache      CacheInterface
	rootDir    *Directory
	mountPoint string
	mu         sync.RWMutex
	stats      FSStats
	startTime  time.Time
}

// ServiceInterface defines the methods needed from the memory service
// This allows us to mock/test without a real service
type ServiceInterface interface {
	SearchMemories(ctx context.Context, req *types.SearchRequest) ([]types.MemoryResult, error)
	GetMemory(ctx context.Context, id string) (*types.Memory, error)
	CreateMemory(ctx context.Context, content string, opts ...interface{}) (*types.Memory, error)
	UpdateMemory(ctx context.Context, id, content string) error
	DeleteMemory(ctx context.Context, id string) error
	ListSkills(ctx context.Context, tenantID string) ([]*types.Skill, error)
	ListSessions(ctx context.Context, userID string) ([]*types.Session, error)
	GetEntity(ctx context.Context, id string) (*types.Entity, error)
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
	TotalMemories  int64
	TotalEntities  int64
	TotalSkills    int64
	TotalSessions int64
	CacheHits     int64
	CacheMisses   int64
	ReadOps       int64
	WriteOps      int64
	DeleteOps     int64
	Uptime       time.Duration
}

// NewVirtualFS creates a new virtual filesystem
// Pattern: like NewService() in memory/service.go
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

func NewVirtualFS(svc ServiceInterface, mountPoint string) *VirtualFS {
	fs := &VirtualFS{
		svc:        svc,
		inodeMgr:   NewInodeManager(100000),
		cache:      newMemoryCache(10000),
		mountPoint: mountPoint,
		rootDir:    newRootDirectory(),
		startTime:  time.Now(),
	}

	fs.buildDirectoryTree()

	return fs
}

// buildDirectoryTree creates the initial directory structure
// Structure:
// /
//   ├── memories/
//   │   └── [user-id]/
//   │       └── [memory-id].md
//   ├── skills/
//   │   └── [skill-name].md
//   ├── sessions/
//   │   └── [session-id]/
//   └── search -> symlink to dynamic results
func (fs *VirtualFS) buildDirectoryTree() {
	// Root directory already created in newRootDirectory()
	// Add top-level directories
	fs.addDir("/", "memories", false)
	fs.addDir("/", "skills", false)
	fs.addDir("/", "sessions", false)
	fs.addDir("/", "entities", false)
}

// addDir adds a directory entry to a parent path
func (fs *VirtualFS) addDir(parentPath, name string, createInode bool) *Directory {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	var parentDir *Directory
	if parentPath == "/" {
		parentDir = fs.rootDir
	} else {
		if _, parentInode, ok := fs.inodeMgr.GetByPath(parentPath); ok {
			if dir, ok := fs.lookupDir(parentInode.ID); ok {
				parentDir = dir
			}
		}
	}

	if parentDir == nil {
		return nil
	}

	if _, ok := parentDir.Children[name]; ok {
		return nil
	}

	var inode *Inode
	if createInode {
		_, inode = fs.inodeMgr.Allocate(parentPath, name, true, "", "")
	}

	dir := &Directory{
		Inode:    inode,
		Parent:   parentDir,
		Children: make(map[string]*DirEntry),
	}

	parentDir.Children[name] = &DirEntry{
		Name:  name,
		Inode: inode.ID,
		IsDir: true,
	}

	return dir
}

// lookupDir finds a directory by inode ID
func (fs *VirtualFS) lookupDir(inodeID uint64) (*Directory, bool) {
	// Simplified: traverse from root
	return nil, false
}

// ReadDir lists directory entries at a path
func (fs *VirtualFS) ReadDir(ctx context.Context, dirPath string) ([]DirEntry, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	fs.stats.ReadOps++

	// Check cache first
	if cached, ok := fs.cache.Get("dir:" + dirPath); ok {
		fs.stats.CacheHits++
		if entries, ok := cached.([]DirEntry); ok {
			return entries, nil
		}
	}
	fs.stats.CacheMisses++

	// Handle special directories
	switch dirPath {
	case "/":
		return fs.readRootDir()
	case "/memories":
		return fs.readMemoriesDir(ctx)
	case "/skills":
		return fs.readSkillsDir(ctx)
	case "/sessions":
		return fs.readSessionsDir(ctx)
	case "/entities":
		return fs.readEntitiesDir(ctx)
	}

	// Handle user-specific memory directories: /memories/[user-id]
	if strings.HasPrefix(dirPath, "/memories/") {
		parts := strings.Split(dirPath, "/")
		if len(parts) >= 3 {
			userID := parts[2]
			return fs.readUserMemoriesDir(ctx, userID)
		}
	}

	// Handle skill files: /skills/[skill-name].md
	if strings.HasPrefix(dirPath, "/skills/") {
		return nil, fmt.Errorf("not a directory: %s", dirPath)
	}

	return nil, fmt.Errorf("directory not found: %s", dirPath)
}

// readRootDir lists root directory entries
func (fs *VirtualFS) readRootDir() ([]DirEntry, error) {
	entries := []DirEntry{
		{Name: "memories", Inode: 1, IsDir: true},
		{Name: "skills", Inode: 2, IsDir: true},
		{Name: "sessions", Inode: 3, IsDir: true},
		{Name: "entities", Inode: 4, IsDir: true},
	}
	return entries, nil
}

// readMemoriesDir lists memories directory
func (fs *VirtualFS) readMemoriesDir(ctx context.Context) ([]DirEntry, error) {
	// List all users (simplified: return static entry)
	// In production, this would query Neo4j for distinct user IDs
	return []DirEntry{
		// Dynamic: would list user directories
	}, nil
}

// readUserMemoriesDir lists memories for a specific user
func (fs *VirtualFS) readUserMemoriesDir(ctx context.Context, userID string) ([]DirEntry, error) {
	// Query memories for this user
	results, err := fs.svc.SearchMemories(ctx, &types.SearchRequest{
		UserID: userID,
		Limit:  1000,
	})
	if err != nil {
		return nil, fmt.Errorf("search memories: %w", err)
	}

	entries := make([]DirEntry, 0, len(results))
	for _, result := range results {
		filename := result.MemoryID + ".md"
		if result.Metadata != nil {
			if result.Metadata.Content != "" {
				filename = result.MemoryID + ".md"
			}
		}

		filePath := path.Join("/memories", userID, filename)
		id, _ := fs.inodeMgr.Allocate(filePath, filename, false, result.MemoryID, "")

		entries = append(entries, DirEntry{
			Name:  filename,
			Inode: id,
			IsDir: false,
		})
	}

	return entries, nil
}

// readSkillsDir lists skills directory
func (fs *VirtualFS) readSkillsDir(ctx context.Context) ([]DirEntry, error) {
	// This would query the skills service
	// Simplified implementation
	return []DirEntry{}, nil
}

// readSessionsDir lists sessions directory
func (fs *VirtualFS) readSessionsDir(ctx context.Context) ([]DirEntry, error) {
	// This would query the session service
	return []DirEntry{}, nil
}

// readEntitiesDir lists entities directory
func (fs *VirtualFS) readEntitiesDir(ctx context.Context) ([]DirEntry, error) {
	return []DirEntry{}, nil
}

// ReadFile reads a file's content at a path
func (fs *VirtualFS) ReadFile(ctx context.Context, filePath string) ([]byte, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	fs.stats.ReadOps++

	// Check cache first
	if cached, ok := fs.cache.Get("file:" + filePath); ok {
		fs.stats.CacheHits++
		if content, ok := cached.([]byte); ok {
			return content, nil
		}
	}
	fs.stats.CacheMisses++

	// Parse path to determine file type
	parts := strings.Split(filePath, "/")
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid file path: %s", filePath)
	}

	// Handle memory files: /memories/[user-id]/[memory-id].md
	if strings.HasPrefix(filePath, "/memories/") && strings.HasSuffix(filePath, ".md") {
		return fs.readMemoryFile(ctx, filePath)
	}

	// Handle skill files: /skills/[skill-name].md
	if strings.HasPrefix(filePath, "/skills/") && strings.HasSuffix(filePath, ".md") {
		return fs.readSkillFile(ctx, filePath)
	}

	return nil, fmt.Errorf("file not found: %s", filePath)
}

// readMemoryFile reads a memory file and formats it as markdown
func (fs *VirtualFS) readMemoryFile(ctx context.Context, filePath string) ([]byte, error) {
	// Extract memory ID from path
	// /memories/[user-id]/[memory-id].md
	parts := strings.Split(filePath, "/")
	if len(parts) < 4 {
		return nil, fmt.Errorf("invalid memory path: %s", filePath)
	}

	// Remove .md extension
	filename := parts[len(parts)-1]
	memoryID := strings.TrimSuffix(filename, ".md")
	if memoryID == "" {
		return nil, fmt.Errorf("invalid memory filename: %s", filename)
	}

	// Fetch memory from service
	mem, err := fs.svc.GetMemory(ctx, memoryID)
	if err != nil {
		return nil, fmt.Errorf("get memory: %w", err)
	}

	// Format as markdown with metadata header
	content := fs.formatMemoryAsMarkdown(mem)

	// Cache the content
	fs.cache.Set("file:"+filePath, []byte(content))

	// Update inode size
	if id, _, ok := fs.inodeMgr.GetByPath(filePath); ok {
		fs.inodeMgr.UpdateSize(id, uint64(len(content)))
	}

	return []byte(content), nil
}

// formatMemoryAsMarkdown formats a memory as a markdown file
// This creates a human-readable file that also embeds metadata
func (fs *VirtualFS) formatMemoryAsMarkdown(mem *types.Memory) string {
	var sb strings.Builder

	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("ID: %s\n", mem.ID))
	sb.WriteString(fmt.Sprintf("UserID: %s\n", mem.UserID))
	sb.WriteString(fmt.Sprintf("CreatedAt: %s\n", mem.CreatedAt.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("Importance: %s\n", mem.Importance))
	if len(mem.Tags) > 0 {
		sb.WriteString(fmt.Sprintf("Tags: %v\n", mem.Tags))
	}
	sb.WriteString("---\n\n")

	sb.WriteString(mem.Content)
	sb.WriteString("\n")

	if mem.EntityID != "" {
		sb.WriteString(fmt.Sprintf("\n## Entity\n- %s\n", mem.EntityID))
	}

	return sb.String()
}

// readSkillFile reads a skill file
func (fs *VirtualFS) readSkillFile(ctx context.Context, filePath string) ([]byte, error) {
	// Placeholder: would fetch skill from service
	return []byte("# Skill\n\nSkill content here.\n"), nil
}

// WriteFile writes content to a file (creates or updates memory)
func (fs *VirtualFS) WriteFile(ctx context.Context, filePath string, content []byte) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	fs.stats.WriteOps++

	// Only support writing to /memories/[user-id]/[memory-id].md
	if !strings.HasPrefix(filePath, "/memories/") || !strings.HasSuffix(filePath, ".md") {
		return fmt.Errorf("write only supported for memory files")
	}

	// Extract user ID and memory ID
	relPath := strings.TrimPrefix(filePath, "/memories/")
	parts := strings.Split(relPath, "/")
	if len(parts) < 2 {
		return fmt.Errorf("invalid memory path: %s", filePath)
	}

	userID := parts[0]
	filename := parts[len(parts)-1]
	memoryID := strings.TrimSuffix(filename, ".md")

	// Parse content: extract metadata and content
	memContent := string(content)
	_ = userID // Use userID if needed

	// If memoryID is "new", create a new memory
	if memoryID == "new" {
		mem, err := fs.svc.CreateMemory(ctx, memContent)
		if err != nil {
			return fmt.Errorf("create memory: %w", err)
		}

		// Allocate inode for new file
		name := mem.ID + ".md"
		fs.inodeMgr.Allocate(path.Join("/memories", userID, name), name, false, mem.ID, "")
		return nil
	}

	// Update existing memory
	err := fs.svc.UpdateMemory(ctx, memoryID, memContent)
	if err != nil {
		return fmt.Errorf("update memory: %w", err)
	}

	// Invalidate cache
	fs.cache.Delete("file:" + filePath)

	return nil
}

// DeleteFile deletes a file (deletes the memory)
func (fs *VirtualFS) DeleteFile(ctx context.Context, filePath string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	fs.stats.DeleteOps++

	// Extract memory ID
	parts := strings.Split(filePath, "/")
	if len(parts) < 4 {
		return fmt.Errorf("invalid memory path: %s", filePath)
	}

	filename := parts[len(parts)-1]
	memoryID := strings.TrimSuffix(filename, ".md")

	// Delete from service
	err := fs.svc.DeleteMemory(ctx, memoryID)
	if err != nil {
		return fmt.Errorf("delete memory: %w", err)
	}

	// Remove inode
	fs.inodeMgr.DeleteByPath(filePath)

	// Invalidate cache
	fs.cache.Delete("file:" + filePath)

	return nil
}

// GetAttr returns file attributes for a path
func (fs *VirtualFS) GetAttr(filePath string) (*FileAttr, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	// Check if path exists
	if id, inode, ok := fs.inodeMgr.GetByPath(filePath); ok {
		_ = id
		return &FileAttr{
			Ino:  inode.ID,
			Size: inode.Size,
			Blocks: (inode.Size + 511) / 512, // Round up to 512-byte blocks
			Atime: uint64(inode.ModTime.Unix()),
			Mtime: uint64(inode.ModTime.Unix()),
			Ctime: uint64(inode.ModTime.Unix()),
			Mode:  inode.Mode,
			Nlink: 1,
			UID:  0,
			GID:  0,
			Rdev: 0,
		}, nil
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
	return fs.inodeMgr.GetByPath(p)
}

func (fs *VirtualFS) AddDir(parentPath, name string) {
	fs.addDir(parentPath, name, true)
}

func newRootDirectory() *Directory {
	return &Directory{
		Inode: &Inode{
			ID:   1,
			Mode:  0o755 | 0o40000, // Directory
			IsDir: true,
			Name:  "/",
			Path:  "/",
		},
		Parent:   nil,
		Children: map[string]*DirEntry{
			"memories": {Name: "memories", Inode: 2, IsDir: true},
			"skills":    {Name: "skills", Inode: 3, IsDir: true},
			"sessions":  {Name: "sessions", Inode: 4, IsDir: true},
			"entities":  {Name: "entities", Inode: 5, IsDir: true},
		},
	}
}
