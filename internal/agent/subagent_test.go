package agent

import (
	"testing"
)

func TestManager_RegisterBuiltIns(t *testing.T) {
	manager := NewManager()

	expectedTypes := []string{
		"general",
		"research",
		"coding",
		"planning",
		"review",
	}

	for _, agentType := range expectedTypes {
		factory, exists := manager.registry[agentType]
		if !exists {
			t.Errorf("expected built-in agent type %q to be registered, but it was not", agentType)
			continue
		}

		if factory == nil {
			t.Errorf("expected factory for agent type %q to be non-nil", agentType)
		} else if factory.Create == nil {
			t.Errorf("expected Create function for agent type %q to be non-nil", agentType)
		}
	}
}
