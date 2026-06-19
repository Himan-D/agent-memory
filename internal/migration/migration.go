package migration

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type Migration struct {
	Version     int
	Description string
	Up          string
	Down        string
}

type Executor interface {
	Run(ctx context.Context, cypher string, params map[string]any) error
	RunRead(ctx context.Context, cypher string, params map[string]any) ([]map[string]any, error)
}

type Migrator struct {
	migrations []Migration
	exec       Executor
}

func NewMigrator(exec Executor) *Migrator {
	return &Migrator{
		migrations: knownMigrations(),
		exec:       exec,
	}
}

func knownMigrations() []Migration {
	return []Migration{
		{
			Version:     1,
			Description: "Initial schema: indexes and constraints",
			Up: `CREATE CONSTRAINT memory_id IF NOT EXISTS FOR (n:Memory) REQUIRE n.id IS UNIQUE;
CREATE CONSTRAINT entity_id IF NOT EXISTS FOR (n:Entity) REQUIRE n.id IS UNIQUE;
CREATE CONSTRAINT session_id IF NOT EXISTS FOR (n:Session) REQUIRE n.id IS UNIQUE;
CREATE CONSTRAINT agent_id IF NOT EXISTS FOR (n:Agent) REQUIRE n.id IS UNIQUE;
CREATE CONSTRAINT skill_id IF NOT EXISTS FOR (n:Skill) REQUIRE n.id IS UNIQUE;
CREATE INDEX memory_content IF NOT EXISTS FOR (n:Memory) ON (n.content);
CREATE INDEX memory_created IF NOT EXISTS FOR (n:Memory) ON (n.created_at);
CREATE INDEX entity_name IF NOT EXISTS FOR (n:Entity) ON (n.name);`,
			Down: `DROP CONSTRAINT memory_id IF EXISTS;
DROP CONSTRAINT entity_id IF EXISTS;
DROP CONSTRAINT session_id IF EXISTS;
DROP CONSTRAINT agent_id IF EXISTS;
DROP CONSTRAINT skill_id IF EXISTS;
DROP INDEX memory_content IF EXISTS;
DROP INDEX memory_created IF EXISTS;
DROP INDEX entity_name IF EXISTS;`,
		},
		{
			Version:     2,
			Description: "Add concept nodes and reminder fields",
			Up: `CREATE CONSTRAINT concept_id IF NOT EXISTS FOR (n:Concept) REQUIRE n.id IS UNIQUE;
CREATE INDEX concept_name IF NOT EXISTS FOR (n:Concept) ON (n.name);
CREATE INDEX reminder_due IF NOT EXISTS FOR (n:Reminder) ON (n.due_at);`,
			Down: `DROP CONSTRAINT concept_id IF EXISTS;
DROP INDEX concept_name IF EXISTS;
DROP INDEX reminder_due IF EXISTS;`,
		},
		{
			Version:     3,
			Description: "Add usage tracking nodes",
			Up: `CREATE CONSTRAINT api_key_id IF NOT EXISTS FOR (n:APIKey) REQUIRE n.id IS UNIQUE;
CREATE INDEX webhook_event IF NOT EXISTS FOR (n:Webhook) ON (n.event);
CREATE INDEX memory_tier IF NOT EXISTS FOR (n:Memory) ON (n.tier);`,
			Down: `DROP CONSTRAINT api_key_id IF EXISTS;
DROP INDEX webhook_event IF EXISTS;
DROP INDEX memory_tier IF EXISTS;`,
		},
		{
			Version:     4,
			Description: "Add notification and feedback indexes",
			Up: `CREATE INDEX notification_created IF NOT EXISTS FOR (n:Notification) ON (n.created_at);
CREATE INDEX feedback_memory IF NOT EXISTS FOR (n:Feedback) ON (n.memory_id);
CREATE INDEX message_session IF NOT EXISTS FOR (n:Message) ON (n.session_id);`,
			Down: `DROP INDEX notification_created IF EXISTS;
DROP INDEX feedback_memory IF EXISTS;
DROP INDEX message_session IF EXISTS;`,
		},
	}
}

func (m *Migrator) GetCurrentVersion(ctx context.Context) (int, error) {
	rows, err := m.exec.RunRead(ctx, "MATCH (v:SchemaVersion) RETURN v.version ORDER BY v.version DESC LIMIT 1", nil)
	if err != nil {
		return 0, nil
	}
	if len(rows) > 0 {
		v, ok := rows[0]["v.version"].(int64)
		if !ok {
			return 0, nil
		}
		return int(v), nil
	}
	return 0, nil
}

func (m *Migrator) RunPending(ctx context.Context) error {
	current, err := m.GetCurrentVersion(ctx)
	if err != nil {
		return fmt.Errorf("migrator: get current version: %w", err)
	}

	for _, mg := range m.migrations {
		if mg.Version <= current {
			continue
		}
		if err := m.apply(ctx, mg); err != nil {
			return fmt.Errorf("migrator: apply v%d: %w", mg.Version, err)
		}
	}
	return nil
}

func (m *Migrator) apply(ctx context.Context, mg Migration) error {
	if err := runStatements(ctx, m.exec, mg.Up); err != nil {
		return fmt.Errorf("execute migration v%d: %w", mg.Version, err)
	}

	if err := m.exec.Run(ctx, `
		MERGE (v:SchemaVersion {id: 'schema_version'})
		SET v.version = $version, v.applied = $applied, v.description = $description
	`, map[string]any{
		"version":     int64(mg.Version),
		"applied":     time.Now().UTC().Format(time.RFC3339),
		"description": mg.Description,
	}); err != nil {
		return fmt.Errorf("record migration v%d: %w", mg.Version, err)
	}

	return nil
}

func (m *Migrator) Rollback(ctx context.Context, targetVersion int) error {
	current, err := m.GetCurrentVersion(ctx)
	if err != nil {
		return fmt.Errorf("migrator: get current version: %w", err)
	}

	for i := len(m.migrations) - 1; i >= 0; i-- {
		mg := m.migrations[i]
		if mg.Version > current || mg.Version <= targetVersion {
			continue
		}
		if err := m.rollbackOne(ctx, mg); err != nil {
			return fmt.Errorf("migrator: rollback v%d: %w", mg.Version, err)
		}
	}
	return nil
}

func (m *Migrator) rollbackOne(ctx context.Context, mg Migration) error {
	if err := runStatements(ctx, m.exec, mg.Down); err != nil {
		return fmt.Errorf("execute rollback v%d: %w", mg.Version, err)
	}

	return m.exec.Run(ctx, `
		MATCH (v:SchemaVersion {id: 'schema_version'})
		SET v.version = $version
	`, map[string]any{"version": int64(mg.Version - 1)})
}

// runStatements splits a multi-statement Cypher string by semicolons and
// executes each statement individually. Neo4j only accepts one statement
// per query, so multi-statement migrations must be split.
//
// When a CREATE CONSTRAINT fails because a plain index already exists on
// the same property, the conflicting index is dropped and the constraint
// is retried. This handles upgrades from versions that used plain indexes
// where the current migration uses uniqueness constraints.
func runStatements(ctx context.Context, exec Executor, cypher string) error {
	stmts := strings.Split(cypher, ";")
	for _, stmt := range stmts {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if err := exec.Run(ctx, stmt, nil); err != nil {
			if strings.Contains(err.Error(), "IndexAlreadyExists") && strings.HasPrefix(stmt, "CREATE CONSTRAINT") {
				if err := dropConflictingIndex(ctx, exec, stmt); err == nil {
					if err := exec.Run(ctx, stmt, nil); err == nil {
						continue
					}
				}
			}
			return fmt.Errorf("%s: %w", stmt, err)
		}
	}
	return nil
}

// dropConflictingIndex parses a CREATE CONSTRAINT statement to extract the
// label and property, then drops any plain index on that label+property.
func dropConflictingIndex(ctx context.Context, exec Executor, constraintStmt string) error {
	// Extract label from "FOR (n:Label)" pattern
	labelStart := strings.Index(constraintStmt, "(:")
	if labelStart == -1 {
		return fmt.Errorf("cannot parse label from constraint")
	}
	labelEnd := strings.Index(constraintStmt[labelStart:], ")")
	if labelEnd == -1 {
		return fmt.Errorf("cannot parse label end")
	}
	label := constraintStmt[labelStart+2 : labelStart+labelEnd]

	// Extract property from "ON (n.prop)" or "REQUIRE n.prop" pattern
	var prop string
	if onIdx := strings.Index(constraintStmt, "ON ("); onIdx != -1 {
		dotIdx := strings.Index(constraintStmt[onIdx:], ".")
		parenEnd := strings.Index(constraintStmt[onIdx:], ")")
		if dotIdx != -1 && parenEnd != -1 {
			prop = constraintStmt[onIdx+dotIdx+1 : onIdx+parenEnd]
		}
	} else if reqIdx := strings.Index(constraintStmt, "REQUIRE "); reqIdx != -1 {
		dotIdx := strings.Index(constraintStmt[reqIdx:], ".")
		spaceIdx := strings.Index(constraintStmt[reqIdx+dotIdx:], " ")
		if dotIdx != -1 && spaceIdx != -1 {
			prop = constraintStmt[reqIdx+dotIdx+1 : reqIdx+dotIdx+spaceIdx]
		}
	}
	if prop == "" {
		return fmt.Errorf("cannot parse property from constraint")
	}

	// Query for existing indexes on this label and drop the conflicting one
	rows, err := exec.RunRead(ctx, "SHOW INDEXES", nil)
	if err != nil {
		return err
	}
	for _, row := range rows {
		labels, _ := row["labelsOrTypes"].([]interface{})
		for _, l := range labels {
			if ls, ok := l.(string); ok && ls == label {
				if name, ok := row["name"].(string); ok && name != "" {
					// Don't drop constraints, only plain indexes
					if idxType, ok := row["type"].(string); ok && strings.Contains(idxType, "CONSTRAINT") {
						continue
					}
					_ = exec.Run(ctx, fmt.Sprintf("DROP INDEX %s IF EXISTS", name), nil)
				}
			}
		}
	}
	return nil
}

func (m *Migrator) Status(ctx context.Context) ([]MigrationStatus, error) {
	current, err := m.GetCurrentVersion(ctx)
	if err != nil {
		return nil, err
	}

	var statuses []MigrationStatus
	for _, mg := range m.migrations {
		statuses = append(statuses, MigrationStatus{
			Version:     mg.Version,
			Description: mg.Description,
			Applied:     mg.Version <= current,
		})
	}
	return statuses, nil
}

type MigrationStatus struct {
	Version     int
	Description string
	Applied     bool
}

type Neo4jExecutor struct {
	RunFn  func(ctx context.Context, cypher string, params map[string]any) error
	ReadFn func(ctx context.Context, cypher string, params map[string]any) ([]map[string]any, error)
}

func (e *Neo4jExecutor) Run(ctx context.Context, cypher string, params map[string]any) error {
	return e.RunFn(ctx, cypher, params)
}

func (e *Neo4jExecutor) RunRead(ctx context.Context, cypher string, params map[string]any) ([]map[string]any, error) {
	return e.ReadFn(ctx, cypher, params)
}
