package unified

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError  `json:"error,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

const ProtocolVersion = "2024-11-05"

func (s *Server) runStdio(ctx context.Context, in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req JSONRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			s.writeResponse(out, JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      nil,
				Error:   &RPCError{Code: -32700, Message: "Parse error"},
			})
			continue
		}

		resp := s.handleRequest(ctx, req)
		s.writeResponse(out, resp)
	}
	return scanner.Err()
}

func (s *Server) runHTTP(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", s.handleHTTP)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	srv := &http.Server{
		Addr:    s.config.Addr,
		Handler: s.corsMiddleware(s.authMiddleware(mux)),
	}

	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx)
		return nil
	case err := <-errCh:
		return err
	}
}

func (s *Server) runSSE(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", s.handleHTTP)
	mux.HandleFunc("/mcp/sse", s.handleSSE)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	srv := &http.Server{
		Addr:    s.config.Addr,
		Handler: s.corsMiddleware(s.authMiddleware(mux)),
	}

	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx)
		return nil
	case err := <-errCh:
		return err
	}
}

func (s *Server) handleHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req JSONRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	if tid := r.Header.Get("X-Tenant-ID"); tid != "" {
		ctx = contextWithTenant(ctx, tid)
	}

	resp := s.handleRequest(ctx, req)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	fmt.Fprintf(w, "event: endpoint\ndata: /mcp\n\n")
	flusher.Flush()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			fmt.Fprintf(w, ": heartbeat\n\n")
			flusher.Flush()
		}
	}
}

func (s *Server) handleRequest(ctx context.Context, req JSONRPCRequest) JSONRPCResponse {
	switch req.Method {
	case "initialize":
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"protocolVersion": ProtocolVersion,
				"capabilities": map[string]interface{}{
					"tools": map[string]interface{}{},
				},
				"serverInfo": map[string]interface{}{
					"name":    "hystersis-memory",
					"version": "1.0.0",
				},
			},
		}
	case "tools/list":
		s.mu.RLock()
		tools := s.tools
		s.mu.RUnlock()

		var toolList []map[string]interface{}
		for _, t := range tools {
			toolList = append(toolList, map[string]interface{}{
				"name":        t.Name,
				"description": t.Description,
				"inputSchema": t.InputSchema,
			})
		}
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  map[string]interface{}{"tools": toolList},
		}
	case "tools/call":
		var params struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   &RPCError{Code: -32602, Message: "Invalid params"},
			}
		}

		s.mu.RLock()
		var handler func(ctx context.Context, args map[string]interface{}) (interface{}, error)
		for _, t := range s.tools {
			if t.Name == params.Name {
				handler = t.Handler
				break
			}
		}
		s.mu.RUnlock()

		if handler == nil {
			return JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   &RPCError{Code: -32601, Message: fmt.Sprintf("Tool not found: %s", params.Name)},
			}
		}

		result, err := handler(ctx, params.Arguments)
		if err != nil {
			return JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   &RPCError{Code: -32603, Message: err.Error()},
			}
		}
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  map[string]interface{}{"content": result},
		}
	default:
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &RPCError{Code: -32601, Message: fmt.Sprintf("Method not found: %s", req.Method)},
		}
	}
	if ctx.Err() != nil {
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &RPCError{Code: -32000, Message: "Context cancelled"},
		}
	}
	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Error:   &RPCError{Code: -32603, Message: "Internal error"},
	}
}

func (s *Server) writeResponse(w io.Writer, resp JSONRPCResponse) {
	data, err := json.Marshal(resp)
	if err != nil {
		log.Printf("mcp: marshal error: %v", err)
		return
	}
	w.Write(data)
	w.Write([]byte("\n"))
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Tenant-ID, X-API-Key")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.config.APIKey != "" {
			apiKey := r.Header.Get("X-API-Key")
			if apiKey == "" {
				apiKey = r.Header.Get("Authorization")
				if len(apiKey) > 7 && apiKey[:7] == "Bearer " {
					apiKey = apiKey[7:]
				}
			}
			if apiKey != s.config.APIKey {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}