package nfs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"agent-memory/internal/fs/vfs"
)

// NFSServer implements an NFS-like server for macOS compatibility
// Pattern: follows server/main.go initialization
type NFSServer struct {
	svc        vfs.ServiceInterface
	exportPath string
	allowList map[string]bool // Allowed export paths
	mu         sync.RWMutex
	stats      NFSSStats
	startTime  time.Time
}

// NFSSStats tracks server statistics
type NFSSStats struct {
	ConnectionsTotal int64
	ExportsTotal     int64
	ReadOps         int64
	WriteOps        int64
	Errors          int64
}

// NFSExport represents an exported directory
type NFSExport struct {
	Path        string
	AllowedHosts []string
	ReadOnly    bool
	MapAllUID   uint32
	MapAllGID   uint32
}

// NewNFSServer creates a new NFS server
// Pattern: like NewCompressionPipeline() in compression/pipeline/
func NewNFSServer(svc vfs.ServiceInterface, exportPath string) *NFSServer {
	return &NFSServer{
		svc:        svc,
		exportPath: exportPath,
		allowList:  make(map[string]bool),
		startTime:  time.Now(),
	}
}

// AddExport adds an export path
func (s *NFSServer) AddExport(path string, readOnly bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.allowList[path] = true
	log.Printf("Added NFS export: %s (readOnly=%v)", path, readOnly)
}

// Start begins serving NFS requests
// Pattern: like Syncer.Start() in sync/syncer.go
func (s *NFSServer) Start(ctx context.Context) error {
	log.Printf("NFS server starting on export path: %s", s.exportPath)

	// In production: implement actual NFS protocol (RFC 1094, RFC 1813)
	// For now, implement as HTTP-based file operations (simplified)

	mux := http.NewServeMux()

	// NFS mount (simplified - uses HTTP POST)
	mux.HandleFunc("/nfs/mount", s.handleMount)
	mux.HandleFunc("/nfs/umount", s.handleUnmount)

	// File operations
	mux.HandleFunc("/nfs/read", s.handleRead)
	mux.HandleFunc("/nfs/write", s.handleWrite)
	mux.HandleFunc("/nfs/readdir", s.handleReaddir)

	// Stats
	mux.HandleFunc("/nfs/stats", s.handleStats)

	server := &http.Server{
		Addr:    ":2049", // Standard NFS port
		Handler: mux,
	}

	// Start server in goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("NFS server error: %v", err)
			s.mu.Lock()
			s.stats.Errors++
			s.mu.Unlock()
		}
	}()

	log.Println("NFS server started on :2049")

	// Wait for context cancellation
	<-ctx.Done()
	return server.Shutdown(context.Background())
}

// handleMount handles NFS mount requests (simplified)
func (s *NFSServer) handleMount(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	s.stats.ConnectionsTotal++
	s.mu.Unlock()

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "path required", http.StatusBadRequest)
		return
	}

	s.mu.RLock()
	_, allowed := s.allowList[path]
	s.mu.RUnlock()

	if !allowed {
		http.Error(w, "path not exported", http.StatusForbidden)
		return
	}

	s.mu.Lock()
	s.stats.ExportsTotal++
	s.mu.Unlock()

	log.Printf("NFS mount: %s", path)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"mounted","path":"%s"}`, path)
}

// handleUnmount handles NFS unmount
func (s *NFSServer) handleUnmount(w http.ResponseWriter, r *http.Request) {
	log.Println("NFS unmount requested")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"status":"unmounted"}`)
}

// handleRead handles file read operations
func (s *NFSServer) handleRead(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	s.stats.ReadOps++
	s.mu.Unlock()

	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		http.Error(w, "path required", http.StatusBadRequest)
		return
	}

	// Read file content (delegates to VirtualFS)
	// In production: would use actual NFS READ procedure
	content, err := s.readFile(filePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(content)
}

// handleWrite handles file write operations
func (s *NFSServer) handleWrite(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	s.stats.WriteOps++
	s.mu.Unlock()

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		http.Error(w, "path required", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Write file content (delegates to VirtualFS)
	// In production: would use actual NFS WRITE procedure
	if err := s.writeFile(filePath, body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"status":"written"}`)
}

// handleReaddir handles directory listing
func (s *NFSServer) handleReaddir(w http.ResponseWriter, r *http.Request) {
	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		filePath = "/"
	}

	// List directory (delegates to VirtualFS)
	entries, err := s.readDir(filePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)
}

// handleStats returns server statistics
func (s *NFSServer) handleStats(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	stats := s.stats
	s.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// readFile reads a file's content
// Pattern: follows VirtualFS.ReadFile()
func (s *NFSServer) readFile(filePath string) ([]byte, error) {
	// In production: call s.svc (VirtualFS)
	// For now, simplified implementation
	if !strings.HasPrefix(filePath, "/memories/") {
		return nil, fmt.Errorf("unsupported path: %s", filePath)
	}

	// Extract memory ID from path
	parts := strings.Split(filePath, "/")
	if len(parts) < 4 {
		return nil, fmt.Errorf("invalid path: %s", filePath)
	}

	// Would call: mem, err := s.svc.GetMemory(ctx, memoryID)
	// Return mem.Content as bytes
	return []byte("# Memory\n\nContent here.\n"), nil
}

// writeFile writes content to a file
func (s *NFSServer) writeFile(filePath string, content []byte) error {
	// In production: call s.svc.CreateMemory() or UpdateMemory()
	log.Printf("NFS write: %s (%d bytes)", filePath, len(content))
	return nil
}

// readDir lists directory entries
func (s *NFSServer) readDir(dirPath string) ([]vfs.DirEntry, error) {
	// In production: call s.svc.ReadDir()
	// Simplified implementation
	return []vfs.DirEntry{
		{Name: "example.md", Inode: 100, IsDir: false},
	}, nil
}
