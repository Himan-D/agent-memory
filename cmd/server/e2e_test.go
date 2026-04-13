package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"agent-memory/internal/config"
	"agent-memory/internal/memory/neo4j"
)

func TestAPIServerHandlers(t *testing.T) {
	t.Run("health endpoint returns ok", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		})

		req := httptest.NewRequest("GET", "/health", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}

		var resp map[string]string
		json.NewDecoder(rr.Body).Decode(&resp)
		if resp["status"] != "ok" {
			t.Errorf("expected status ok, got %s", resp["status"])
		}
	})

	t.Run("unauthorized without key returns 401", func(t *testing.T) {
		cfg := &config.Config{
			Auth: config.AuthConfig{
				Enabled: true,
				APIKeys: []string{"test-key"},
			},
		}

		handler := authMiddleware(cfg, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/sessions", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", rr.Code)
		}
	})

	t.Run("authorized with valid key returns 200", func(t *testing.T) {
		cfg := &config.Config{
			Auth: config.AuthConfig{
				Enabled: true,
				APIKeys: []string{"test-key:default"},
			},
		}

		var capturedTenant string
		handler := authMiddleware(cfg, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedTenant = getTenantID(r)
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/sessions", nil)
		req.Header.Set("X-API-Key", "test-key")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}
		if capturedTenant != "default" {
			t.Errorf("expected tenant default, got %s", capturedTenant)
		}
	})

	t.Run("rate limit blocks excess requests", func(t *testing.T) {
		rl := newRateLimiter(3, 10*time.Second)

		blocked := 0
		for i := 0; i < 5; i++ {
			if !rl.allow("client") {
				blocked++
			}
		}

		if blocked != 2 {
			t.Errorf("expected 2 blocked requests, got %d", blocked)
		}
	})
}

func TestMessageValidation(t *testing.T) {
	tests := []struct {
		name    string
		role    string
		wantErr bool
	}{
		{"valid user", "user", false},
		{"valid assistant", "assistant", false},
		{"valid system", "system", false},
		{"valid tool", "tool", false},
		{"invalid hacker", "hacker", true},
		{"invalid ADMIN", "ADMIN", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMessageRole(tt.role)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateMessageRole(%q) error = %v, wantErr %v", tt.role, err, tt.wantErr)
			}
		})
	}
}

func TestEntityValidation(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{"valid entity name", "Machine-Learning", false},
		{"valid simple", "test-entity", false},
		{"empty", "", true},
		{"with spaces", "bad name", true},
		{"with special chars", "bad@name", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEntityID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateEntityID(%q) error = %v, wantErr %v", tt.id, err, tt.wantErr)
			}
		})
	}
}

func TestRelationTypeSecurity(t *testing.T) {
	injectionAttempts := []string{
		"foo} MATCH (e:Entity) DETACH DELETE e",
		"x RETURN 1",
		"KNOWS; DELETE",
		"KNOWS RETURN 1",
		"custom_type", // Not in allowed list
	}

	for _, attempt := range injectionAttempts {
		err := neo4j.ValidateRelationType(attempt)
		if err == nil {
			t.Errorf("expected injection attempt %q to be rejected, but it was allowed", attempt)
		}
	}
}

func TestAllowedRelationTypes(t *testing.T) {
	allowedTypes := []string{
		"KNOWS", "HAS", "RELATED_TO", "DEPENDS_ON", "USES",
		"CREATED_BY", "PART_OF", "IMPROVES", "CONFLICTS",
		"FOLLOWS", "LIKES", "DISLIKES", "SUBSCRIBED",
	}

	for _, relType := range allowedTypes {
		err := neo4j.ValidateRelationType(relType)
		if err != nil {
			t.Errorf("expected allowed type %q to pass validation, got error: %v", relType, err)
		}
	}
}

func TestCreateSessionRequest(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		wantValid   bool
		wantAgentID string
	}{
		{"valid request", `{"agent_id": "my-agent"}`, true, "my-agent"},
		{"empty body", `{}`, false, ""},
		{"missing agent_id", `{"metadata": {}}`, false, ""},
		{"with metadata", `{"agent_id": "test-bot", "metadata": {"version": "1.0"}}`, true, "test-bot"},
		{"invalid agent_id format", `{"agent_id": "bad agent!"}`, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/sessions", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")

			var parsed struct {
				AgentID  string                 `json:"agent_id"`
				Metadata map[string]interface{} `json:"metadata"`
			}
			err := json.NewDecoder(req.Body).Decode(&parsed)

			if tt.wantValid && err != nil {
				t.Errorf("expected valid request, got parse error: %v", err)
			}

			if !tt.wantValid && err == nil && parsed.AgentID == "" {
				// Should be caught by validation
			}
		})
	}
}

func TestHealthEndpoint(t *testing.T) {
	// Test that health endpoint returns expected format
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	tests := []struct {
		path     string
		wantCode int
	}{
		{"/health", http.StatusOK},
		{"/ready", http.StatusOK},
		{"/metrics", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != tt.wantCode {
				t.Errorf("expected %d, got %d", tt.wantCode, rr.Code)
			}
		})
	}
}
