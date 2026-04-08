package main

import (
	"testing"
)

func TestValidateAgentID(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{"valid simple", "my-agent", false},
		{"valid with underscore", "my_agent_123", false},
		{"valid with dash", "my-agent-123", false},
		{"valid max length", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", false},
		{"empty", "", true},
		{"too long", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", true},
		{"invalid chars", "my agent", true},
		{"special chars", "my@agent", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAgentID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateAgentID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateEntityID(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{"valid", "Machine-Learning", false},
		{"empty", "", true},
		{"invalid chars", "entity name", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEntityID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateEntityID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateMessageRole(t *testing.T) {
	tests := []struct {
		name    string
		role    string
		wantErr bool
	}{
		{"valid user", "user", false},
		{"valid assistant", "assistant", false},
		{"valid system", "system", false},
		{"valid tool", "tool", false},
		{"empty", "", true},
		{"invalid", "hacker", true},
		{"uppercase", "USER", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMessageRole(tt.role)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateMessageRole() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
