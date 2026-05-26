package migration

import "time"

// Migration describes a single versioned schema change.
type Migration struct {
	Version     int
	Description string
	Applied     time.Time
}

// Migrator tracks and applies pending schema migrations.
type Migrator struct {
	migrations []Migration
}

// NewMigrator returns a Migrator pre-loaded with all known migrations.
func NewMigrator() *Migrator {
	return &Migrator{
		migrations: []Migration{
			{Version: 1, Description: "Initial schema - indexes and constraints"},
			{Version: 2, Description: "Add concept nodes and reminder fields"},
			{Version: 3, Description: "Add usage tracking nodes"},
		},
	}
}

// GetCurrentVersion returns the current schema version recorded in Neo4j.
// A return value of 0 means no migrations have been applied yet.
func (m *Migrator) GetCurrentVersion() int {
	// TODO: read (:SchemaVersion {version}) node from Neo4j.
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
		// TODO: execute the DDL/Cypher for mg.Version against Neo4j and
		// upsert the (:SchemaVersion) node with the new version number.
	}
	return nil
}
