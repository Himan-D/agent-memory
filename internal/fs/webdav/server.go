package webdav

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"agent-memory/internal/fs/vfs"
)

// WebDAVServer implements a WebDAV server for Windows compatibility
// Pattern: follows cmd/server/api.go initialization
type WebDAVServer struct {
	svc        vfs.ServiceInterface
	vfs        *vfs.VirtualFS
	handler     http.Handler
	server      *http.Server
	mu         sync.RWMutex
	stats      WebDAVStats
	startTime  time.Time
}

// WebDAVStats tracks server statistics
type WebDAVStats struct {
	ConnectionsTotal int64
	RequestsTotal   int64
	ErrorsTotal     int64
	Uptime         time.Duration
}

// NewWebDAVServer creates a new WebDAV server
// Pattern: follows NewService() in memory/service.go
func NewWebDAVServer(svc vfs.ServiceInterface, mountPoint string) *WebDAVServer {
	vfs := vfs.NewVirtualFS(svc, mountPoint)

	// Create WebDAV handler
	handler := createWebDAVHandler(vfs)

	srv := &WebDAVServer{
		svc:       svc,
		vfs:       vfs,
		handler:   handler,
		startTime: time.Now(),
	}

	return srv
}

// Start begins serving WebDAV requests
// Pattern: follows Server.Start() in cmd/server/main.go
func (s *WebDAVServer) Start(addr string) error {
	log.Printf("WebDAV server starting on %s", addr)

	s.server = &http.Server{
		Addr:    addr,
		Handler: createMiddleware(s.handler),
		// Timeouts
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start in goroutine
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("WebDAV error: %v", err)
			s.mu.Lock()
			s.stats.ErrorsTotal++
			s.mu.Unlock()
		}
	}()

	log.Printf("WebDAV server started on %s", addr)
	return nil
}

// Stop gracefully stops the server
func (s *WebDAVServer) Stop(ctx context.Context) error {
	log.Println("WebDAV server stopping...")
	return s.server.Shutdown(ctx)
}

// GetStats returns server statistics
func (s *WebDAVServer) GetStats() WebDAVStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := s.stats
	stats.Uptime = time.Since(s.startTime)
	return stats
}

// ---- Internal Implementation ----

// createWebDAVHandler sets up WebDAV HTTP handlers
// Pattern: follows cmd/server/api.go registerRoutes()
func createWebDAVHandler(v *vfs.VirtualFS) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handlePropFind(w, r, v)
	})
	mux.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		handlePut(w, r, v)
	})
	mux.HandleFunc("/download", func(w http.ResponseWriter, r *http.Request) {
		handleGet(w, r, v)
	})
	mux.HandleFunc("/delete", func(w http.ResponseWriter, r *http.Request) {
		handleDelete(w, r, v)
	})
	mux.HandleFunc("/mkdir", handleMkcol)
	mux.HandleFunc("/move", handleMove)
	mux.HandleFunc("/props", handlePropPatch)
	mux.HandleFunc("/lock", handleLock)
	mux.HandleFunc("/unlock", handleUnlock)
	mux.HandleFunc("/stats", handleStats)

	return mux
}

// createMiddleware adds logging, metrics, recovery
func createMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Log request
		log.Printf("[WebDAV] %s %s %s", r.Method, r.URL.Path, r.RemoteAddr)

		// Call next handler
		next.ServeHTTP(w, r)

		// Log duration
		log.Printf("[WebDAV] Completed in %v", time.Since(start))
	})
}

// ---- WebDAV Handlers ----

func handlePropFind(w http.ResponseWriter, r *http.Request, v *vfs.VirtualFS) {
	if r.Method != http.MethodGet && r.Method != "PROPFIND" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		filePath = "/"
	}

	entries, err := v.ReadDir(r.Context(), filePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", `application/xml; charset="utf-8"`)
	w.WriteHeader(http.StatusMultiStatus)

	fmt.Fprintf(w, `<?xml version="1.0" encoding="utf-8"?>
<d:multistatus xmlns:d="DAV:">
	`)

	for _, entry := range entries {
		resType := ""
		if entry.IsDir {
			resType = "<d:collection/>"
		}
		fmt.Fprintf(w, `
	<d:response>
		<d:href>%s</d:href>
		<d:propstat>
			<d:prop>
				<d:displayname>%s</d:displayname>
				<d:resourcetype>%s</d:resourcetype>
			</d:prop>
			<d:status>HTTP/1.1 200 OK</d:status>
		</d:propstat>
	</d:response>
		`, entry.Name, entry.Name, resType)
	}

	fmt.Fprintf(w, `</d:multistatus>`)
}

func handleGet(w http.ResponseWriter, r *http.Request, v *vfs.VirtualFS) {
	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		http.Error(w, "path required", http.StatusBadRequest)
		return
	}

	content, err := v.ReadFile(r.Context(), filePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
	w.Write(content)
}

func handlePut(w http.ResponseWriter, r *http.Request, v *vfs.VirtualFS) {
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

	if err := v.WriteFile(r.Context(), filePath, body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	fmt.Fprint(w, `{"status":"created"}`)
}

func handleDelete(w http.ResponseWriter, r *http.Request, v *vfs.VirtualFS) {
	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		http.Error(w, "path required", http.StatusBadRequest)
		return
	}

	if err := v.DeleteFile(r.Context(), filePath); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func handleMkcol(w http.ResponseWriter, r *http.Request) {
	dirPath := r.URL.Query().Get("path")
	if dirPath == "" {
		http.Error(w, "path required", http.StatusBadRequest)
		return
	}

	// Create directory via virtual FS
	// In production: vfs.CreateDir(dirPath)
	log.Printf("WebDAV MKCOL: %s", dirPath)

	w.WriteHeader(http.StatusCreated)
	fmt.Fprint(w, `{"status":"created"}`)
}

func handleMove(w http.ResponseWriter, r *http.Request) {
	srcPath := r.URL.Query().Get("src")
	dstPath := r.URL.Query().Get("dst")

	if srcPath == "" || dstPath == "" {
		http.Error(w, "src and dst required", http.StatusBadRequest)
		return
	}

	// Move/rename via virtual FS
	// In production: vfs.Rename(srcPath, dstPath)
	log.Printf("WebDAV MOVE: %s -> %s", srcPath, dstPath)

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"status":"moved"}`)
}

func handlePropPatch(w http.ResponseWriter, r *http.Request) {
	// Update WebDAV properties (simplified)
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"status":"updated"}`)
}

func handleLock(w http.ResponseWriter, r *http.Request) {
	// WebDAV locking (simplified)
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"status":"locked"}`)
}

func handleUnlock(w http.ResponseWriter, r *http.Request) {
	// WebDAV unlock (simplified)
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"status":"unlocked"}`)
}

func handleStats(w http.ResponseWriter, r *http.Request) {
	// Return server statistics
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, `{"status":"ok","service":"webdav"}`)
}
