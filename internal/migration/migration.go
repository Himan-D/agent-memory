package migration

import (
	"context"
	"fmt"
	"time"

	neo4jdriver "github.com/neo4j/neo4j-go-driver/v6/neo4j"
)

// Migration describes a single versioned schema change.
type Migration struct {
	Version     int
	Description string
	Applied     time.Time
	Cypher      []string
}

// Migrator tracks and applies pending schema migrations against Neo4j.
type Migrator struct {
	driver     neo4jdriver.DriverWithContext
	migrations []Migration
}

// NewMigrator returns a Migrator pre-loaded with all known migrations.
func NewMigrator(driver neo4jdriver.DriverWithContext) *Migrator {
	return &Migrator{
		driver: driver,
		migrations: []Migration{
			{
				Version:     1,
				Description: "Initial schema - indexes and constraints",
				Cypher: []string{
					"CREATE INDEX entity_id_idx IF NOT EXISTS FOR (e:Entity) ON (e.id)",
					"CREATE INDEX session_id_idx IF NOT EXISTS FOR (s:Session) ON (s.id)",
					"CREATE INDEX memory_user_idx IF NOT EXISTS FOR (m:Memory) ON (m.user_id)",
				},
			},
			{
				Version:     2,
				Description: "Add concept nodes and reminder fields",
				Cypher: []string{
					"CREATE INDEX concept_name_idx IF NOT EXISTS FOR (c:Concept) ON (c.name)",
				},
			},
			{
				Version:     3,
				Description: "Add usage tracking nodes",
				Cypher: []string{
					"CREATE INDEX usage_agent_idx IF NOT EXISTS FOR (u:UsageRecord) ON (u.agent_id)",
				},
			},
		},
	}
}

// GetCurrentVersion returns the current schema version recorded in Neo4j.
// A return value of 0 means no migrations have been applied yet.
func (m *Migrator) GetCurrentVersion() int {
	ctx := context.Background()
	session := m.driver.NewSession(ctx, neo4jdriver.SessionConfig{AccessMode: neo4jdriver.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.Run(ctx,
		"MATCH (s:SchemaVersion) RETURN s.version AS version",
		nil,
	)
	if err != nil {
		return 0
	}

	if result.Next(ctx) {
		record := result.Record()
		if v, ok := record.Get("version"); ok && v != nil {
			switch val := v.(type) {
			case int64:
				return int(val)
			case int:
				return val
			}
		}
	}

	return 0
}

// RunPending applies every migration whose version is greater than the current
// schema version. It is idempotent — migrations already applied are skipped.
func (m *Migrator) RunPending() error {
	current := m.GetCurrentVersion()

	for _, mg := range m.migrations {
		if mg.Version <= current {
			continue
		}

		ctx := context.Background()
		session := m.driver.NewSession(ctx, neo4jdriver.SessionConfig{AccessMode: neo4jdriver.AccessModeWrite})

		for _, cypher := range mg.Cypher {
			_, err := session.Run(ctx, cypher, nil)
			if err != nil {
				session.Close(ctx)
				return fmt.Errorf("migration v%d: execute cypher: %w", mg.Version, err)
			}
		}

		_, err := session.Run(ctx,
			"MERGE (s:SchemaVersion) SET s.version = $version, s.applied_at = datetime()",
			map[string]any{"version": mg.Version},
		)
		if err != nil {
			session.Close(ctx)
			return fmt.Errorf("migration v%d: upsert schema version: %w", mg.Version, err)
		}

		session.Close(ctx)
	}

	return nil
}
