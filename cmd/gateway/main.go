package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"syscall"
	"time"
)

var (
	port             = flag.String("port", "8080", "Gateway port")
	memoryAPIURL     = flag.String("memory-api", "http://localhost:8081", "Memory API URL")
	mcpServerURL      = flag.String("mcp-server", "http://localhost:8082", "MCP Server URL")
	connectorsURL     = flag.String("connectors", "http://localhost:8083", "Connectors URL")
	enableTryFiles   = flag.Bool("try-files", true, "Enable SPA try-files")
	enablePrometheus = flag.Bool("prometheus", true, "Enable metrics")
)

type ProxyConfig struct {
	Target   string
	Rewrite  *regexp.Regexp
	Headers  map[string]string
}

type Gateway struct {
	memoryAPIURL string
	mcpServerURL string
	connectorsURL string
	httpServer   *http.Server
	proxies      map[string]*ProxyConfig
}

func NewGateway() *Gateway {
	proxies := map[string]*ProxyConfig{
		"/api/v1/": {
			Target: *memoryAPIURL,
			Headers: map[string]string{
				"X-Service": "memory-api",
			},
		},
		"/memories": {
			Target: *memoryAPIURL,
			Headers: map[string]string{
				"X-Service": "memory-api",
			},
		},
		"/search": {
			Target: *memoryAPIURL,
			Headers: map[string]string{
				"X-Service": "memory-api",
			},
		},
		"/api/v1/benchmark": {
			Target: *memoryAPIURL,
			Headers: map[string]string{
				"X-Service": "memory-api",
			},
		},
		"/mcp": {
			Target: *mcpServerURL,
			Headers: map[string]string{
				"X-Service": "mcp-server",
			},
		},
		"/oauth": {
			Target: *mcpServerURL,
			Headers: map[string]string{
				"X-Service": "mcp-server",
			},
		},
		"/.well-known": {
			Target: *mcpServerURL,
			Headers: map[string]string{
				"X-Service": "mcp-server",
			},
		},
		"/connectors": {
			Target: *connectorsURL,
			Headers: map[string]string{
				"X-Service": "connectors",
			},
		},
	}

	mux := http.NewServeMux()
	
	// Health endpoints
	mux.HandleFunc("/health", handleGatewayHealth)
	mux.HandleFunc("/ready", handleGatewayReady)
	
	// Metrics (simplified)
	if *enablePrometheus {
		mux.HandleFunc("/metrics", handleMetrics)
	}

	// Proxy all other requests
	for path, config := range proxies {
		mux.HandleFunc(path, createProxyHandler(path, config))
	}

	// Add catch-all for SPA
	if *enableTryFiles {
		mux.HandleFunc("/", handleSPA)
	}

	httpServer := &http.Server{
		Addr:         *port,
		Handler:      loggingMiddleware(mux),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return &Gateway{
		memoryAPIURL: *memoryAPIURL,
		mcpServerURL: *mcpServerURL,
		connectorsURL: *connectorsURL,
		httpServer:   httpServer,
		proxies:      proxies,
	}
}

func (g *Gateway) Start() error {
	log.Printf("Gateway starting on %s", *port)
	log.Printf("Route mappings:")
	log.Printf("  /api/v1/*     -> %s", g.memoryAPIURL)
	log.Printf("  /memories     -> %s", g.memoryAPIURL)
	log.Printf("  /search       -> %s", g.memoryAPIURL)
	log.Printf("  /mcp/*        -> %s", g.mcpServerURL)
	log.Printf("  /oauth/*      -> %s", g.mcpServerURL)
	log.Printf("  /connectors/* -> %s", g.connectorsURL)

	if err := g.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("Gateway: %w", err)
	}
	return nil
}

func (g *Gateway) Stop(ctx context.Context) error {
	log.Println("Gateway shutting down...")
	if err := g.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	return nil
}

func main() {
	flag.Parse()

	gateway := NewGateway()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutdown signal, stopping gateway...")
		gateway.Stop(ctx)
		cancel()
	}()

	if err := gateway.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// ==================== Handlers ====================

func handleGatewayHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{
		"status": "ok",
		"service": "gateway",
		"version": "1.0.0"
	}`))
}

func handleGatewayReady(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	// Check upstream services
	_, memErr := http.Get(*memoryAPIURL + "/health")
	_, mcpErr := http.Get(*mcpServerURL + "/health")
	_, conErr := http.Get(*connectorsURL + "/health")
	
	ready := memErr == nil && mcpErr == nil && conErr == nil
	
	if ready {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ready"}`))
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"status": "not ready", "errors": ["memory-api", "mcp-server", "connectors"]}`))
	}
}

func handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("# Gateway metrics\ngateway_requests_total 1\n"))
}

func handleSPA(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`<!DOCTYPE html><html><body><h1>Hystersis API Gateway</h1></body></html>`))
}

// ==================== Proxy ====================

func createProxyHandler(path string, config *ProxyConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Build upstream URL
		upstream := config.Target + r.URL.Path
		
		// Create proxy request
		req, _ := http.NewRequest(r.Method, upstream, r.Body)
		req.Header = make(http.Header)
		
		// Copy headers
		for k, v := range r.Header {
			req.Header[k] = v
		}
		
		// Add service headers
		for k, v := range config.Headers {
			req.Header.Set(k, v)
		}

		// Make request
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Printf("Proxy error: %v", err)
			http.Error(w, "Upstream error", http.StatusBadGateway)
			return
		}
		
		// Copy response
		for k, v := range resp.Header {
			w.Header().Set(k, v[0])
		}
		
		w.WriteHeader(resp.StatusCode)
		
		buf := make([]byte, resp.ContentLength+100)
		resp.Body.Read(buf)
		w.Write(buf)
		
		log.Printf("%s %s -> %d (%v)", r.Method, r.URL.Path, resp.StatusCode, time.Since(start))
	}
}

// ==================== Middleware ====================

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
	})
}