package neo4j

import (
	"testing"
	"time"

	neo4jdriver "github.com/neo4j/neo4j-go-driver/v6/neo4j"
)

func TestRecordToAgentPreservesConfig(t *testing.T) {
	client := &Client{}
	now := time.Now()

	agent, err := client.recordToAgent(&neo4jdriver.Record{Values: []interface{}{
		"agent-1",
		"tenant-1",
		"Support Agent",
		"Answers support requests",
		"active",
		`{"auto_extract":true,"sharing_policy":"team","skill_domains":["support","billing"]}`,
		`["group-1"]`,
		`{"region":"us"}`,
		now,
		now,
		nil,
	}})
	if err != nil {
		t.Fatalf("recordToAgent returned error: %v", err)
	}

	if !agent.Config.AutoExtract {
		t.Fatalf("expected auto_extract to be preserved")
	}
	if agent.Config.SharingPolicy != "team" {
		t.Fatalf("expected sharing policy team, got %q", agent.Config.SharingPolicy)
	}
	if len(agent.Config.SkillDomains) != 2 || agent.Config.SkillDomains[0] != "support" || agent.Config.SkillDomains[1] != "billing" {
		t.Fatalf("expected skill domains to be preserved, got %#v", agent.Config.SkillDomains)
	}
}

func TestRecordToAgentGroupPreservesPolicyAndMembers(t *testing.T) {
	client := &Client{}
	now := time.Now()

	group, err := client.recordToAgentGroup(&neo4jdriver.Record{Values: []interface{}{
		"group-1",
		"tenant-1",
		"Support Team",
		"Customer support",
		"support",
		`{"allow_cross_agent_memory":true,"skill_sharing_enabled":false,"require_human_review":true}`,
		"pool-1",
		`{"region":"us"}`,
		now,
		now,
		[]interface{}{
			map[string]interface{}{
				"agent_id":  "agent-1",
				"role":      "member",
				"joined_at": now,
			},
		},
	}})
	if err != nil {
		t.Fatalf("recordToAgentGroup returned error: %v", err)
	}

	if !group.Policy.AllowCrossAgentMemory {
		t.Fatalf("expected allow_cross_agent_memory to be preserved")
	}
	if group.Policy.SkillSharingEnabled {
		t.Fatalf("expected skill_sharing_enabled=false to be preserved")
	}
	if !group.Policy.RequireHumanReview {
		t.Fatalf("expected require_human_review to be preserved")
	}
	if len(group.Members) != 1 || group.Members[0].AgentID != "agent-1" {
		t.Fatalf("expected members to be read from the members column, got %#v", group.Members)
	}
}
