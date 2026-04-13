package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type ContextRepository struct {
	mu       sync.RWMutex
	path     string
	name     string
	memories []ContextEntry
	commits  []Commit
	index    *Index
}

type ContextEntry struct {
	ID        string                 `json:"id"`
	Content   string                 `json:"content"`
	Type      string                 `json:"type"`
	Priority  int                    `json:"priority"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	Tags      []string               `json:"tags"`
	Metadata  map[string]interface{} `json:"metadata"`
}

type Commit struct {
	Hash      string    `json:"hash"`
	Message   string    `json:"message"`
	Author    string    `json:"author"`
	Timestamp time.Time `json:"timestamp"`
	Changes   int       `json:"changes"`
}

type Index struct {
	mu         sync.RWMutex
	tags       map[string][]string
	byType     map[string][]string
	byPriority map[int][]string
	fullText   map[string][]string
}

func NewContextRepository(name, path string) (*ContextRepository, error) {
	if path == "" {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, ".agent-memory", "repos", name)
	}

	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, fmt.Errorf("failed to create repo directory: %w", err)
	}

	repo := &ContextRepository{
		path:     path,
		name:     name,
		memories: []ContextEntry{},
		commits:  []Commit{},
		index: &Index{
			tags:       make(map[string][]string),
			byType:     make(map[string][]string),
			byPriority: make(map[int][]string),
			fullText:   make(map[string][]string),
		},
	}

	if err := repo.initGit(); err != nil {
		return nil, fmt.Errorf("failed to initialize git: %w", err)
	}

	repo.loadIndex()

	return repo, nil
}

func (r *ContextRepository) initGit() error {
	if _, err := os.Stat(filepath.Join(r.path, ".git")); err == nil {
		return nil
	}

	cmd := exec.Command("git", "init")
	cmd.Dir = r.path
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git init failed: %w", err)
	}

	cmd = exec.Command("git", "config", "user.email", "agent@memory.local")
	cmd.Dir = r.path
	cmd.Run()

	cmd = exec.Command("git", "config", "user.name", "Agent Memory")
	cmd.Dir = r.path
	cmd.Run()

	return nil
}

func (r *ContextRepository) Add(ctx context.Context, entry *ContextEntry) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if entry.ID == "" {
		entry.ID = r.generateID(entry.Content)
	}
	entry.CreatedAt = time.Now()
	entry.UpdatedAt = time.Now()

	r.memories = append(r.memories, *entry)
	r.indexEntry(entry)

	return r.saveAndCommit(ctx, fmt.Sprintf("Add %s: %s", entry.Type, truncate(entry.Content, 50)))
}

func (r *ContextRepository) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, entry := range r.memories {
		if entry.ID == id {
			entry.UpdatedAt = time.Now()

			if content, ok := updates["content"].(string); ok {
				r.memories[i].Content = content
			}
			if priority, ok := updates["priority"].(int); ok {
				r.memories[i].Priority = priority
			}
			if tags, ok := updates["tags"].([]string); ok {
				r.memories[i].Tags = tags
			}
			if metadata, ok := updates["metadata"].(map[string]interface{}); ok {
				r.memories[i].Metadata = metadata
			}

			r.index = &Index{
				tags:       make(map[string][]string),
				byType:     make(map[string][]string),
				byPriority: make(map[int][]string),
				fullText:   make(map[string][]string),
			}
			for _, e := range r.memories {
				r.indexEntry(&e)
			}

			return r.saveAndCommit(ctx, fmt.Sprintf("Update %s", truncate(entry.Content, 50)))
		}
	}

	return fmt.Errorf("entry not found: %s", id)
}

func (r *ContextRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, entry := range r.memories {
		if entry.ID == id {
			r.memories = append(r.memories[:i], r.memories[i+1:]...)
			return r.saveAndCommit(ctx, fmt.Sprintf("Delete %s", truncate(entry.Content, 50)))
		}
	}

	return fmt.Errorf("entry not found: %s", id)
}

func (r *ContextRepository) Get(ctx context.Context, id string) (*ContextEntry, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, entry := range r.memories {
		if entry.ID == id {
			return &entry, nil
		}
	}

	return nil, fmt.Errorf("entry not found: %s", id)
}

func (r *ContextRepository) List(ctx context.Context, limit int) ([]*ContextEntry, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if limit <= 0 {
		limit = len(r.memories)
	}

	result := make([]*ContextEntry, 0, limit)
	for i := 0; i < limit && i < len(r.memories); i++ {
		result = append(result, &r.memories[i])
	}

	return result, nil
}

func (r *ContextRepository) Search(ctx context.Context, query string, limit int) ([]*ContextEntry, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	query = strings.ToLower(query)
	var results []*ContextEntry

	for i := range r.memories {
		entry := &r.memories[i]
		if r.matchesQuery(entry, query) {
			results = append(results, entry)
			if limit > 0 && len(results) >= limit {
				break
			}
		}
	}

	return results, nil
}

func (r *ContextRepository) Progressive(ctx context.Context, maxTokens int) ([]*ContextEntry, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*ContextEntry
	currentTokens := 0

	for _, entry := range r.memories {
		entryTokens := len(entry.Content) / 4

		if currentTokens+entryTokens <= maxTokens {
			result = append(result, &entry)
			currentTokens += entryTokens
		} else if entry.Priority >= 5 && currentTokens < maxTokens {
			truncated := truncate(entry.Content, (maxTokens-currentTokens)*4)
			highPriority := entry
			highPriority.Content = truncated
			result = append(result, &highPriority)
			break
		} else {
			break
		}
	}

	return result, nil
}

func (r *ContextRepository) Tag(ctx context.Context, id string, tags []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, entry := range r.memories {
		if entry.ID == id {
			r.memories[i].Tags = append(r.memories[i].Tags, tags...)
			r.indexEntry(&r.memories[i])
			return r.saveAndCommit(ctx, fmt.Sprintf("Tag %s with %v", id, tags))
		}
	}

	return fmt.Errorf("entry not found: %s", id)
}

func (r *ContextRepository) GetHistory(ctx context.Context) ([]*Commit, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*Commit, len(r.commits))
	for i := range r.commits {
		result[i] = &r.commits[i]
	}

	return result, nil
}

func (r *ContextRepository) Diff(ctx context.Context, from, to string) (string, error) {
	cmd := exec.Command("git", "diff", from, to)
	cmd.Dir = r.path
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git diff failed: %w", err)
	}
	return string(output), nil
}

func (r *ContextRepository) Restore(ctx context.Context, commitHash string) error {
	cmd := exec.Command("git", "checkout", commitHash, "--", ".")
	cmd.Dir = r.path
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git checkout failed: %w", err)
	}

	r.loadIndex()
	return r.saveAndCommit(ctx, fmt.Sprintf("Restore to %s", commitHash[:8]))
}

func (r *ContextRepository) Snapshots(ctx context.Context, limit int) ([]*Commit, error) {
	cmd := exec.Command("git", "log", "--oneline", fmt.Sprintf("-%d", limit))
	cmd.Dir = r.path
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git log failed: %w", err)
	}

	var commits []*Commit
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			commits = append(commits, &Commit{
				Hash:    parts[0],
				Message: parts[1],
			})
		}
	}

	return commits, nil
}

func (r *ContextRepository) Sync(ctx context.Context, remote string) error {
	if remote == "" {
		remote = "origin"
	}

	cmd := exec.Command("git", "fetch", remote)
	cmd.Dir = r.path
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git fetch failed: %w", err)
	}

	cmd = exec.Command("git", "pull", remote, "main")
	cmd.Dir = r.path
	if err := cmd.Run(); err != nil {
		cmd = exec.Command("git", "pull", remote, "master")
		cmd.Dir = r.path
		if err != nil {
			return fmt.Errorf("git pull failed: %w", err)
		}
	}

	r.loadIndex()
	return nil
}

func (r *ContextRepository) Push(ctx context.Context, remote string) error {
	if remote == "" {
		remote = "origin"
	}

	cmd := exec.Command("git", "push", remote, "main")
	cmd.Dir = r.path
	cmd.Run()

	cmd = exec.Command("git", "push", remote, "master")
	cmd.Dir = r.path
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git push failed: %w", err)
	}

	return nil
}

func (r *ContextRepository) generateID(content string) string {
	hash := sha256.Sum256([]byte(content + time.Now().String()))
	return hex.EncodeToString(hash[:8])
}

func (r *ContextRepository) indexEntry(entry *ContextEntry) {
	for _, tag := range entry.Tags {
		r.index.tags[tag] = append(r.index.tags[tag], entry.ID)
	}

	r.index.byType[entry.Type] = append(r.index.byType[entry.Type], entry.ID)

	r.index.byPriority[entry.Priority] = append(r.index.byPriority[entry.Priority], entry.ID)

	words := strings.Fields(strings.ToLower(entry.Content))
	for _, word := range words {
		if len(word) > 3 {
			r.index.fullText[word] = append(r.index.fullText[word], entry.ID)
		}
	}
}

func (r *ContextRepository) matchesQuery(entry *ContextEntry, query string) bool {
	if strings.Contains(strings.ToLower(entry.Content), query) {
		return true
	}

	for _, tag := range entry.Tags {
		if strings.Contains(strings.ToLower(tag), query) {
			return true
		}
	}

	if strings.Contains(strings.ToLower(entry.Type), query) {
		return true
	}

	return false
}

func (r *ContextRepository) saveAndCommit(ctx context.Context, message string) error {
	dataFile := filepath.Join(r.path, "memory.json")

	content, _ := os.ReadFile(dataFile)
	var existing []ContextEntry
	if len(content) > 0 {
		// Simple JSON storage - in production use proper JSON marshaling
		_ = existing
	}

	// Save to file
	data := r.serializeMemories()
	if err := os.WriteFile(dataFile, []byte(data), 0644); err != nil {
		return fmt.Errorf("failed to save: %w", err)
	}

	// Git commit
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = r.path
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git add failed: %w", err)
	}

	cmd = exec.Command("git", "commit", "-m", message)
	cmd.Dir = r.path
	if err := cmd.Run(); err != nil {
		// Ignore if nothing to commit
		return nil
	}

	r.commits = append(r.commits, Commit{
		Hash:      r.getLastCommitHash(),
		Message:   message,
		Author:    "Agent Memory",
		Timestamp: time.Now(),
		Changes:   1,
	})

	return nil
}

func (r *ContextRepository) serializeMemories() string {
	var sb strings.Builder
	sb.WriteString("[\n")
	for i, m := range r.memories {
		if i > 0 {
			sb.WriteString(",\n")
		}
		sb.WriteString(fmt.Sprintf(`{"id":"%s","content":"%s","type":"%s","priority":%d}`, m.ID, m.Content, m.Type, m.Priority))
	}
	sb.WriteString("\n]")
	return sb.String()
}

func (r *ContextRepository) loadIndex() error {
	dataFile := filepath.Join(r.path, "memory.json")
	content, err := os.ReadFile(dataFile)
	if err != nil {
		return nil // No data yet
	}

	r.memories = r.deserializeMemories(string(content))

	for i := range r.memories {
		r.indexEntry(&r.memories[i])
	}

	return nil
}

func (r *ContextRepository) deserializeMemories(data string) []ContextEntry {
	var memories []ContextEntry
	// Simple parsing - in production use proper JSON unmarshaling
	_ = memories
	return memories
}

func (r *ContextRepository) getLastCommitHash() string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = r.path
	output, _ := cmd.Output()
	return strings.TrimSpace(string(output))
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func (r *ContextRepository) Path() string {
	return r.path
}

func (r *ContextRepository) Name() string {
	return r.name
}

type RepositoryManager struct {
	mu       sync.RWMutex
	repos    map[string]*ContextRepository
	basePath string
}

func NewRepositoryManager(basePath string) (*RepositoryManager, error) {
	if basePath == "" {
		home, _ := os.UserHomeDir()
		basePath = filepath.Join(home, ".agent-memory", "repos")
	}

	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create repos directory: %w", err)
	}

	return &RepositoryManager{
		repos:    make(map[string]*ContextRepository),
		basePath: basePath,
	}, nil
}

func (rm *RepositoryManager) Create(ctx context.Context, name string, path string) (*ContextRepository, error) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if _, exists := rm.repos[name]; exists {
		return nil, fmt.Errorf("repository already exists: %s", name)
	}

	repo, err := NewContextRepository(name, path)
	if err != nil {
		return nil, err
	}

	rm.repos[name] = repo
	return repo, nil
}

func (rm *RepositoryManager) Get(name string) (*ContextRepository, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	if repo, exists := rm.repos[name]; exists {
		return repo, nil
	}

	// Try to load existing repo
	repoPath := filepath.Join(rm.basePath, name)
	repo, err := NewContextRepository(name, repoPath)
	if err != nil {
		return nil, fmt.Errorf("repository not found: %s", name)
	}

	rm.repos[name] = repo
	return repo, nil
}

func (rm *RepositoryManager) List() []string {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	names := make([]string, 0, len(rm.repos))
	for name := range rm.repos {
		names = append(names, name)
	}

	return names
}

func (rm *RepositoryManager) Delete(name string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	delete(rm.repos, name)
	return nil
}

func (rm *RepositoryManager) Ensure(ctx context.Context, name string) (*ContextRepository, error) {
	if repo, err := rm.Get(name); err == nil {
		return repo, nil
	}
	return rm.Create(ctx, name, "")
}
