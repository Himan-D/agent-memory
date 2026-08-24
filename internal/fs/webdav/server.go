package webdav

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"

	"agent-memory/internal/fs/vfs"
)

// Server is a minimal WebDAV (RFC 4918) server over AgentFS VirtualFS.
// Usable on macOS via Finder → Connect to Server → http://localhost:8081/
type Server struct {
	vfs       *vfs.VirtualFS
	server    *http.Server
	mu        sync.RWMutex
	startTime time.Time
}

// NewServer creates a WebDAV server bound to a VirtualFS.
func NewServer(v *vfs.VirtualFS) *Server {
	return &Server{vfs: v, startTime: time.Now()}
}

// Start listens on addr (e.g. ":8081").
func (s *Server) Start(addr string) error {
	mux := http.NewServeMux()
	mux.Handle("/", s)
	s.server = &http.Server{
		Addr:         addr,
		Handler:      logging(mux),
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	}
	log.Printf("WebDAV agentfs listening on %s", addr)
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("WebDAV error: %v", err)
		}
	}()
	return nil
}

// Stop shuts down the server.
func (s *Server) Stop(ctx context.Context) error {
	if s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}

// ServeHTTP routes WebDAV methods by path.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p := cleanURLPath(r.URL.Path)
	switch r.Method {
	case "OPTIONS":
		w.Header().Set("Allow", "OPTIONS, GET, HEAD, PUT, DELETE, PROPFIND, MKCOL")
		w.Header().Set("DAV", "1, 2")
		w.WriteHeader(http.StatusOK)
	case "PROPFIND":
		s.propfind(w, r, p)
	case http.MethodGet, http.MethodHead:
		s.get(w, r, p)
	case http.MethodPut:
		s.put(w, r, p)
	case http.MethodDelete:
		s.del(w, r, p)
	case "MKCOL":
		// Virtual dirs are fixed; succeed for known roots
		w.WriteHeader(http.StatusCreated)
	case http.MethodPost:
		// treat as put for convenience
		s.put(w, r, p)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) propfind(w http.ResponseWriter, r *http.Request, p string) {
	depth := r.Header.Get("Depth")
	if depth == "" {
		depth = "1"
	}

	type prop struct {
		DisplayName  string `xml:"D:displayname"`
		ResourceType string `xml:"D:resourcetype"`
		GetContentL  string `xml:"D:getcontentlength,omitempty"`
	}
	type propstat struct {
		Prop   prop   `xml:"D:prop"`
		Status string `xml:"D:status"`
	}
	type response struct {
		HRef     string   `xml:"D:href"`
		PropStat propstat `xml:"D:propstat"`
	}
	type multistatus struct {
		XMLName   xml.Name   `xml:"D:multistatus"`
		Xmlns     string     `xml:"xmlns:D,attr"`
		Responses []response `xml:"D:response"`
	}

	ms := multistatus{Xmlns: "DAV:"}
	// self
	isDir := strings.HasSuffix(p, "/") || p == "/" || isVirtualDir(p)
	selfHref := p
	if isDir && !strings.HasSuffix(selfHref, "/") {
		selfHref += "/"
	}
	rt := ""
	if isDir {
		rt = "<D:collection/>"
	}
	ms.Responses = append(ms.Responses, response{
		HRef: selfHref,
		PropStat: propstat{
			Prop:   prop{DisplayName: path.Base(p), ResourceType: rt},
			Status: "HTTP/1.1 200 OK",
		},
	})

	if depth != "0" && isDir {
		entries, err := s.vfs.ReadDir(r.Context(), p)
		if err == nil {
			for _, e := range entries {
				href := path.Join(p, e.Name)
				if e.IsDir {
					href += "/"
				}
				ert := ""
				if e.IsDir {
					ert = "<D:collection/>"
				}
				ms.Responses = append(ms.Responses, response{
					HRef: href,
					PropStat: propstat{
						Prop:   prop{DisplayName: e.Name, ResourceType: ert},
						Status: "HTTP/1.1 200 OK",
					},
				})
			}
		}
	}

	w.Header().Set("Content-Type", `application/xml; charset="utf-8"`)
	w.WriteHeader(http.StatusMultiStatus)
	fmt.Fprint(w, xml.Header)
	_ = xml.NewEncoder(w).Encode(ms)
}

func (s *Server) get(w http.ResponseWriter, r *http.Request, p string) {
	if isVirtualDir(p) {
		// Redirect dir listing as simple HTML
		entries, err := s.vfs.ReadDir(r.Context(), p)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, "<html><body><h1>%s</h1><ul>", p)
		for _, e := range entries {
			href := path.Join(p, e.Name)
			fmt.Fprintf(w, `<li><a href="%s">%s</a></li>`, href, e.Name)
		}
		fmt.Fprint(w, "</ul></body></html>")
		return
	}
	data, err := s.vfs.ReadFile(r.Context(), p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	if r.Method == http.MethodHead {
		return
	}
	_, _ = w.Write(data)
}

func (s *Server) put(w http.ResponseWriter, r *http.Request, p string) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.vfs.WriteFile(r.Context(), p, body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) del(w http.ResponseWriter, r *http.Request, p string) {
	if err := s.vfs.DeleteFile(r.Context(), p); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func isVirtualDir(p string) bool {
	switch cleanURLPath(p) {
	case "/", "/memories", "/skills", "/sessions", "/entities", "/search", "/archive":
		return true
	default:
		return false
	}
}

func cleanURLPath(p string) string {
	if p == "" {
		return "/"
	}
	p = path.Clean("/" + strings.TrimPrefix(p, "/"))
	return p
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("[webdav] %s %s %v", r.Method, r.URL.Path, time.Since(start))
	})
}

// NewWebDAVServer is an alias for backwards compatibility with older call sites.
func NewWebDAVServer(svc vfs.ServiceInterface, mountPoint string) *Server {
	v := vfs.NewVirtualFS(svc, mountPoint)
	return NewServer(v)
}
