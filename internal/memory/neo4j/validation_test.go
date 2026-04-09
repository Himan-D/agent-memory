package neo4j

import (
	"testing"
)

func TestValidateRelationType(t *testing.T) {
	tests := []struct {
		name    string
		relType string
		wantErr bool
	}{
		{"valid KNOWS", "KNOWS", false},
		{"valid HAS", "HAS", false},
		{"valid RELATED_TO", "RELATED_TO", false},
		{"valid DEPENDS_ON", "DEPENDS_ON", false},
		{"invalid lowercase", "knows", true},
		{"invalid spaces", "KNOWS USES", true},
		{"invalid special", "KNOWS;DELETE", true},
		{"invalid empty", "", true},
		{"invalid inject attempt", "foo} MATCH (e:Entity) DETACH DELETE e", true},
		{"invalid inject 2", "x RETURN 1", true},
		{"not in allowed list", "CUSTOM_RELATION", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRelationType(tt.relType)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRelationType() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAllowedRelTypes(t *testing.T) {
	allowed := map[string]bool{
		"KNOWS":      true,
		"HAS":        true,
		"RELATED_TO": true,
		"DEPENDS_ON": true,
		"USES":       true,
		"CREATED_BY": true,
		"PART_OF":    true,
		"IMPROVES":   true,
		"CONFLICTS":  true,
		"FOLLOWS":    true,
		"LIKES":      true,
		"DISLIKES":   true,
		"SUBSCRIBED": true,
	}

	for relType := range allowed {
		t.Run(relType, func(t *testing.T) {
			if err := ValidateRelationType(relType); err != nil {
				t.Errorf("expected %s to be valid, got error: %v", relType, err)
			}
		})
	}
}
