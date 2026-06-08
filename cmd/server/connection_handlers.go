package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"agent-memory/internal/connections"
)

func (s *APIServer) createConnectionHandler(w http.ResponseWriter, r *http.Request) {
	if s.connectionsSvc == nil {
		jsonError(w, "connections service not initialized", http.StatusServiceUnavailable)
		return
	}
	provider := mux.Vars(r)["provider"]
	var req connections.CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	conn, err := s.connectionsSvc.Create(r.Context(), provider, req)
	if err != nil {
		safeHTTPError(w, r, fmt.Errorf("create connection: %w", err), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(conn)
}

func (s *APIServer) listConnectionsHandler(w http.ResponseWriter, r *http.Request) {
	if s.connectionsSvc == nil {
		jsonError(w, "connections service not initialized", http.StatusServiceUnavailable)
		return
	}
	conns, err := s.connectionsSvc.List(r.Context(), connections.Scope{
		UserID: r.URL.Query().Get("user_id"),
		OrgID:  r.URL.Query().Get("org_id"),
	})
	if err != nil {
		safeHTTPError(w, r, fmt.Errorf("list connections: %w", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"connections": conns,
		"count":       len(conns),
		"providers":   connections.SupportedProviders(),
	})
}

func (s *APIServer) getConnectionHandler(w http.ResponseWriter, r *http.Request) {
	if s.connectionsSvc == nil {
		jsonError(w, "connections service not initialized", http.StatusServiceUnavailable)
		return
	}
	conn, err := s.connectionsSvc.Get(r.Context(), mux.Vars(r)["connectionID"])
	if err != nil {
		safeHTTPError(w, r, fmt.Errorf("get connection: %w", err), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(conn)
}

func (s *APIServer) syncConnectionHandler(w http.ResponseWriter, r *http.Request) {
	if s.connectionsSvc == nil {
		jsonError(w, "connections service not initialized", http.StatusServiceUnavailable)
		return
	}
	var req connections.SyncRequest
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
			return
		}
	}
	result, err := s.connectionsSvc.Sync(r.Context(), mux.Vars(r)["connectionID"], req)
	if err != nil {
		status := http.StatusBadRequest
		if result != nil && result.Status == connections.StatusError {
			status = http.StatusInternalServerError
		}
		safeHTTPError(w, r, fmt.Errorf("sync connection: %w", err), status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *APIServer) deleteConnectionHandler(w http.ResponseWriter, r *http.Request) {
	if s.connectionsSvc == nil {
		jsonError(w, "connections service not initialized", http.StatusServiceUnavailable)
		return
	}
	deleteDocuments, _ := strconv.ParseBool(r.URL.Query().Get("delete_documents"))
	connectionID := mux.Vars(r)["connectionID"]
	if err := s.connectionsSvc.Delete(r.Context(), connectionID, deleteDocuments); err != nil {
		safeHTTPError(w, r, fmt.Errorf("delete connection: %w", err), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":           "deleted",
		"connection_id":    connectionID,
		"delete_documents": deleteDocuments,
	})
}
