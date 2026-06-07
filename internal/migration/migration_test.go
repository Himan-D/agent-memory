package migration

import (
	"context"
	"testing"
)

type mockExecutor struct {
	runFn  func(ctx context.Context, cypher string, params map[string]any) error
	readFn func(ctx context.Context, cypher string, params map[string]any) ([]map[string]any, error)
}

func (m *mockExecutor) Run(ctx context.Context, cypher string, params map[string]any) error {
	return m.runFn(ctx, cypher, params)
}

func (m *mockExecutor) RunRead(ctx context.Context, cypher string, params map[string]any) ([]map[string]any, error) {
	return m.readFn(ctx, cypher, params)
}

func TestNewMigrator(t *testing.T) {
	exec := &mockExecutor{
		runFn: func(ctx context.Context, cypher string, params map[string]any) error { return nil },
		readFn: func(ctx context.Context, cypher string, params map[string]any) ([]map[string]any, error) {
			return nil, nil
		},
	}
	m := NewMigrator(exec)
	if m == nil {
		t.Fatal("expected non-nil migrator")
	}
	if len(m.migrations) != 4 {
		t.Errorf("expected 4 known migrations, got %d", len(m.migrations))
	}
}

func TestGetCurrentVersion_NoSchema(t *testing.T) {
	exec := &mockExecutor{
		runFn: func(ctx context.Context, cypher string, params map[string]any) error { return nil },
		readFn: func(ctx context.Context, cypher string, params map[string]any) ([]map[string]any, error) {
			return nil, nil
		},
	}
	m := NewMigrator(exec)
	v, err := m.GetCurrentVersion(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != 0 {
		t.Errorf("expected version 0, got %d", v)
	}
}

func TestGetCurrentVersion_Existing(t *testing.T) {
	exec := &mockExecutor{
		runFn: func(ctx context.Context, cypher string, params map[string]any) error { return nil },
		readFn: func(ctx context.Context, cypher string, params map[string]any) ([]map[string]any, error) {
			return []map[string]any{
				{"v.version": int64(2)},
			}, nil
		},
	}
	m := NewMigrator(exec)
	v, err := m.GetCurrentVersion(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != 2 {
		t.Errorf("expected version 2, got %d", v)
	}
}

func TestRunPending_All(t *testing.T) {
	var applied int
	exec := &mockExecutor{
		runFn: func(ctx context.Context, cypher string, params map[string]any) error {
			applied++
			return nil
		},
		readFn: func(ctx context.Context, cypher string, params map[string]any) ([]map[string]any, error) {
			return nil, nil
		},
	}
	m := NewMigrator(exec)
	if err := m.RunPending(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if applied != 8 {
		t.Errorf("expected 8 runs (4 migrations + 4 version records), got %d", applied)
	}
}

func TestRunPending_SomeApplied(t *testing.T) {
	var applied int
	exec := &mockExecutor{
		runFn: func(ctx context.Context, cypher string, params map[string]any) error {
			applied++
			return nil
		},
		readFn: func(ctx context.Context, cypher string, params map[string]any) ([]map[string]any, error) {
			return []map[string]any{
				{"v.version": int64(2)},
			}, nil
		},
	}
	m := NewMigrator(exec)
	if err := m.RunPending(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if applied != 4 {
		t.Errorf("expected 4 runs (2 migrations + 2 version records), got %d", applied)
	}
}

func TestRunPending_ApplyError(t *testing.T) {
	exec := &mockExecutor{
		runFn: func(ctx context.Context, cypher string, params map[string]any) error {
			return nil
		},
		readFn: func(ctx context.Context, cypher string, params map[string]any) ([]map[string]any, error) {
			return nil, nil
		},
	}
	m := NewMigrator(exec)
	m.exec = &mockExecutor{
		runFn: func(ctx context.Context, cypher string, params map[string]any) error {
			return nil
		},
		readFn: func(ctx context.Context, cypher string, params map[string]any) ([]map[string]any, error) {
			return nil, nil
		},
	}
}

func TestRunPending_ApplyError_New(t *testing.T) {
	count := 0
	exec := &mockExecutor{
		runFn: func(ctx context.Context, cypher string, params map[string]any) error {
			count++
			if count == 1 {
				return nil
			}
			return nil
		},
		readFn: func(ctx context.Context, cypher string, params map[string]any) ([]map[string]any, error) {
			return nil, nil
		},
	}
	m := NewMigrator(exec)
	if err := m.RunPending(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStatus(t *testing.T) {
	exec := &mockExecutor{
		runFn: func(ctx context.Context, cypher string, params map[string]any) error { return nil },
		readFn: func(ctx context.Context, cypher string, params map[string]any) ([]map[string]any, error) {
			return []map[string]any{
				{"v.version": int64(2)},
			}, nil
		},
	}
	m := NewMigrator(exec)
	statuses, err := m.Status(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(statuses) != 4 {
		t.Errorf("expected 4 statuses, got %d", len(statuses))
	}
	expected := []struct {
		version int
		applied bool
	}{
		{1, true},
		{2, true},
		{3, false},
		{4, false},
	}
	for i, s := range statuses {
		if s.Version != expected[i].version {
			t.Errorf("status[%d]: expected version %d, got %d", i, expected[i].version, s.Version)
		}
		if s.Applied != expected[i].applied {
			t.Errorf("status[%d]: expected applied %v, got %v", i, expected[i].applied, s.Applied)
		}
	}
}

func TestRollback(t *testing.T) {
	var rolls int
	exec := &mockExecutor{
		runFn: func(ctx context.Context, cypher string, params map[string]any) error {
			rolls++
			return nil
		},
		readFn: func(ctx context.Context, cypher string, params map[string]any) ([]map[string]any, error) {
			return []map[string]any{
				{"v.version": int64(4)},
			}, nil
		},
	}
	m := NewMigrator(exec)
	if err := m.Rollback(context.Background(), 2); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rolls != 4 {
		t.Errorf("expected 4 rollback runs (2 downs + 2 version updates), got %d", rolls)
	}
}

func TestNeo4jExecutor(t *testing.T) {
	e := &Neo4jExecutor{
		RunFn: func(ctx context.Context, cypher string, params map[string]any) error { return nil },
		ReadFn: func(ctx context.Context, cypher string, params map[string]any) ([]map[string]any, error) {
			return nil, nil
		},
	}
	if err := e.Run(context.Background(), "", nil); err != nil {
		t.Errorf("unexpected run error: %v", err)
	}
	if _, err := e.RunRead(context.Background(), "", nil); err != nil {
		t.Errorf("unexpected read error: %v", err)
	}
}

func TestKnownMigrations(t *testing.T) {
	migs := knownMigrations()
	if len(migs) != 4 {
		t.Fatalf("expected 4 migrations, got %d", len(migs))
	}
	for i, m := range migs {
		if m.Version != i+1 {
			t.Errorf("migration[%d]: expected version %d, got %d", i, i+1, m.Version)
		}
		if m.Description == "" {
			t.Errorf("migration[%d]: missing description", i)
		}
		if m.Up == "" {
			t.Errorf("migration[%d]: missing Up", i)
		}
		if m.Down == "" {
			t.Errorf("migration[%d]: missing Down", i)
		}
	}
}

func TestMigrator_NilExecutor(t *testing.T) {
	m := &Migrator{
		migrations: knownMigrations(),
		exec:       nil,
	}
	if m == nil {
		t.Fatal("expected non-nil migrator with nil executor")
	}
}
