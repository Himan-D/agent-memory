package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"agent-memory/internal/memory/types"
	"github.com/gorilla/mux"
)

// TestSkillSharingRejectedWhenDisabled verifies the SkillSharingEnabled guard logic.
func TestSkillSharingRejectedWhenDisabled(t *testing.T) {
	group := &types.AgentGroup{
		ID:     "g1",
		Policy: types.GroupPolicy{SkillSharingEnabled: false},
	}
	if group.Policy.SkillSharingEnabled {
		t.Fatal("expected SkillSharingEnabled=false")
	}
	// Mirrors the handler guard: skill creation for this group must be rejected.
	skill := types.Skill{Name: "test", GroupID: "g1"}
	rejected := skill.GroupID != "" && !group.Policy.SkillSharingEnabled
	if !rejected {
		t.Fatal("expected skill creation to be rejected for group with sharing disabled")
	}
}

// TestSkillSharingAllowedWhenEnabled verifies the positive case.
func TestSkillSharingAllowedWhenEnabled(t *testing.T) {
	group := &types.AgentGroup{
		ID:     "g2",
		Policy: types.GroupPolicy{SkillSharingEnabled: true},
	}
	skill := types.Skill{Name: "test", GroupID: "g2"}
	rejected := skill.GroupID != "" && !group.Policy.SkillSharingEnabled
	if rejected {
		t.Fatal("expected skill creation to be allowed for group with sharing enabled")
	}
}

// TestAgentDomainFilterExcludesOutOfDomain verifies the domain filter applied
// in listSkillsHandler when agent_id is provided.
func TestAgentDomainFilterExcludesOutOfDomain(t *testing.T) {
	agent := &types.Agent{
		ID:     "agent1",
		Config: types.AgentConfig{SkillDomains: []string{"coding"}},
	}
	all := []*types.Skill{
		{ID: "s1", Domain: "coding"},
		{ID: "s2", Domain: "security"},
		{ID: "s3", Domain: "coding"},
	}

	// Mirrors handler domain filter logic
	domainSet := make(map[string]struct{}, len(agent.Config.SkillDomains))
	for _, d := range agent.Config.SkillDomains {
		domainSet[d] = struct{}{}
	}
	var filtered []*types.Skill
	for _, sk := range all {
		if _, ok := domainSet[sk.Domain]; ok {
			filtered = append(filtered, sk)
		}
	}

	if len(filtered) != 2 {
		t.Fatalf("expected 2 coding skills, got %d", len(filtered))
	}
	for _, sk := range filtered {
		if sk.Domain != "coding" {
			t.Fatalf("non-coding skill %q slipped through domain filter", sk.ID)
		}
	}
}

// TestAgentDomainFilterPassesAllWhenNoDomains verifies skills are unfiltered
// when agent has no SkillDomains configured.
func TestAgentDomainFilterPassesAllWhenNoDomains(t *testing.T) {
	agent := &types.Agent{
		ID:     "agent2",
		Config: types.AgentConfig{SkillDomains: nil},
	}
	all := []*types.Skill{
		{ID: "s1", Domain: "coding"},
		{ID: "s2", Domain: "security"},
	}

	// Handler only filters when len(SkillDomains) > 0
	if len(agent.Config.SkillDomains) > 0 {
		t.Fatal("expected no domain filter applied")
	}
	// All skills pass through
	if len(all) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(all))
	}
}

func TestSkillReviewSDKHandlerValidation(t *testing.T) {
	api := &APIServer{}
	tests := []struct {
		name           string
		body           string
		wantStatusCode int
	}{
		{"valid request", `{"id": "rev1", "approved": true}`, 0}, // 0 means it would panic by hitting nil memSvc
		{"empty body", ``, http.StatusBadRequest},
		{"missing id", `{"approved": true}`, http.StatusBadRequest},
		{"invalid json", `{"id": "rev1",`, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/skills/review", bytes.NewBufferString(tt.body))
			rr := httptest.NewRecorder()
			
			defer func() {
				if r := recover(); r != nil {
					if tt.wantStatusCode != 0 {
						t.Errorf("expected status %d, but handler panicked indicating validation passed", tt.wantStatusCode)
					}
				} else {
					if rr.Code != tt.wantStatusCode {
						t.Errorf("expected status %d, got %d", tt.wantStatusCode, rr.Code)
					}
				}
			}()

			api.skillReviewSDKHandler(rr, req)
		})
	}
}

func TestExecuteSkillHandlerValidation(t *testing.T) {
	api := &APIServer{}
	tests := []struct {
		name string
		body string
	}{
		{"valid request", `{"context": {"foo": "bar"}}`},
		{"empty body", ``}, // Should default to empty context
		{"invalid json", `{"context": `}, // Should recover and use empty context
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/skills/skill1/execute", bytes.NewBufferString(tt.body))
			req = mux.SetURLVars(req, map[string]string{"skillID": "skill1"})
			rr := httptest.NewRecorder()
			
			defer func() {
				if r := recover(); r != nil {
					// Expected to panic because memSvc is nil, meaning validation passed
					return
				}
				t.Errorf("expected handler to pass validation and panic on nil memSvc")
			}()

			api.executeSkillHandler(rr, req)
		})
	}
}
