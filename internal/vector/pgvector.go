package vector

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strings"
	"time"

	_ "github.com/lib/pq"

	"agent-memory/internal/config"
	"agent-memory/internal/memory/types"
)

type PGVectorClient struct {
	db         *sql.DB
	tableName  string
	vectorSize int
	cfg        config.PGVectorConfig
}

func NewPGVectorClient(cfg config.PGVectorConfig) (*PGVectorClient, error) {
	db, err := sql.Open("postgres", cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("pgvector open: %w", err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("pgvector ping: %w", err)
	}
	c := &PGVectorClient{
		db:         db,
		tableName:  cfg.TableName,
		vectorSize: cfg.VectorSize,
		cfg:        cfg,
	}
	if err := c.ensureSchema(context.Background()); err != nil {
		db.Close()
		return nil, fmt.Errorf("pgvector ensure schema: %w", err)
	}
	return c, nil
}

func (c *PGVectorClient) ensureSchema(ctx context.Context) error {
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `CREATE EXTENSION IF NOT EXISTS vector`)
	if err != nil {
		return fmt.Errorf("create vector extension: %w", err)
	}
	createSQL := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		id TEXT PRIMARY KEY,
		text TEXT NOT NULL DEFAULT '',
		entity_id TEXT NOT NULL DEFAULT '',
		tenant_id TEXT NOT NULL DEFAULT '',
		embedding vector(%d),
		metadata JSONB NOT NULL DEFAULT '{}',
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		last_accessed TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`, c.tableName, c.vectorSize)
	if _, err := tx.ExecContext(ctx, createSQL); err != nil {
		return fmt.Errorf("create table: %w", err)
	}
	idxSQL := fmt.Sprintf(
		`CREATE INDEX IF NOT EXISTS %s_embedding_idx ON %s USING hnsw (embedding vector_cosine_ops)`,
		c.tableName, c.tableName,
	)
	if _, err := tx.ExecContext(ctx, idxSQL); err != nil {
		return fmt.Errorf("create hnsw index: %w", err)
	}
	tenantIdxSQL := fmt.Sprintf(
		`CREATE INDEX IF NOT EXISTS %s_tenant_idx ON %s (tenant_id)`,
		c.tableName, c.tableName,
	)
	if _, err := tx.ExecContext(ctx, tenantIdxSQL); err != nil {
		return fmt.Errorf("create tenant index: %w", err)
	}
	return tx.Commit()
}

func (c *PGVectorClient) StoreEmbedding(ctx context.Context, text string, id string, embedding []float32, metadata map[string]interface{}) (string, error) {
	docID := fmt.Sprintf("%d", time.Now().UnixNano())
	vecStr := float32SliceToVector(embedding)
	metaJSON := "{}"
	if len(metadata) > 0 {
		parts := make([]string, 0, len(metadata))
		for k, v := range metadata {
			parts = append(parts, fmt.Sprintf(`"%s":"%v"`, k, v))
		}
		metaJSON = "{" + strings.Join(parts, ",") + "}"
	}
	insertSQL := fmt.Sprintf(
		`INSERT INTO %s (id, text, entity_id, embedding, metadata, created_at, last_accessed) VALUES ($1, $2, $3, $4::vector, $5::jsonb, NOW(), NOW())`,
		c.tableName,
	)
	_, err := c.db.ExecContext(ctx, insertSQL, docID, text, id, vecStr, metaJSON)
	if err != nil {
		return "", fmt.Errorf("store embedding: %w", err)
	}
	return docID, nil
}

func (c *PGVectorClient) Search(ctx context.Context, query []float32, limit int, threshold float32, filters map[string]interface{}) ([]types.MemoryResult, error) {
	return c.searchWithFilters(ctx, query, limit, threshold, filters)
}

func (c *PGVectorClient) SearchWithTenant(ctx context.Context, query []float32, limit int, threshold float32, filters map[string]interface{}, tenantID string) ([]types.MemoryResult, error) {
	if tenantID != "" {
		if filters == nil {
			filters = make(map[string]interface{})
		}
		filters["tenant_id"] = tenantID
	}
	return c.searchWithFilters(ctx, query, limit, threshold, filters)
}

func (c *PGVectorClient) searchWithFilters(ctx context.Context, query []float32, limit int, threshold float32, filters map[string]interface{}) ([]types.MemoryResult, error) {
	vecStr := float32SliceToVector(query)
	args := []interface{}{vecStr, limit, float64(threshold)}
	argIdx := 3
	whereClause := ""
	if len(filters) > 0 {
		var conditions []string
		for k, v := range filters {
			argIdx++
			conditions = append(conditions, fmt.Sprintf("metadata->>'%s' = $%d", k, argIdx))
			args = append(args, fmt.Sprintf("%v", v))
		}
		whereClause = " AND " + strings.Join(conditions, " AND ")
	}
	selectSQL := fmt.Sprintf(
		`SELECT id, text, entity_id, tenant_id, metadata, 1 - (embedding <=> $1::vector) AS score
		 FROM %s
		 WHERE 1 - (embedding <=> $1::vector) >= $3%s
		 ORDER BY embedding <=> $1::vector
		 LIMIT $2`,
		c.tableName, whereClause,
	)
	rows, err := c.db.QueryContext(ctx, selectSQL, args...)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	defer rows.Close()
	var results []types.MemoryResult
	for rows.Next() {
		var id, text, entityID, tenantID string
		var metaJSON string
		var score float64
		if err := rows.Scan(&id, &text, &entityID, &tenantID, &metaJSON, &score); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		props := map[string]interface{}{
			"entity_id":  entityID,
			"tenant_id":  tenantID,
			"metadata":   metaJSON,
		}
		results = append(results, types.MemoryResult{
			Entity: types.Entity{
				ID:         entityID,
				Properties: props,
			},
			Score:  float32(score),
			Text:   text,
			Source: "pgvector",
		})
	}
	return results, rows.Err()
}

func (c *PGVectorClient) UpdateMemory(ctx context.Context, id string, text string, metadata map[string]interface{}) error {
	metaJSON := "{}"
	if len(metadata) > 0 {
		parts := make([]string, 0, len(metadata))
		for k, v := range metadata {
			parts = append(parts, fmt.Sprintf(`"%s":"%v"`, k, v))
		}
		metaJSON = "{" + strings.Join(parts, ",") + "}"
	}
	updateSQL := fmt.Sprintf(
		`UPDATE %s SET text = $2, metadata = $3::jsonb, last_accessed = NOW() WHERE id = $1`,
		c.tableName,
	)
	_, err := c.db.ExecContext(ctx, updateSQL, id, text, metaJSON)
	if err != nil {
		return fmt.Errorf("update memory: %w", err)
	}
	return nil
}

func (c *PGVectorClient) DeleteMemory(ctx context.Context, id string) error {
	deleteSQL := fmt.Sprintf(`DELETE FROM %s WHERE id = $1`, c.tableName)
	_, err := c.db.ExecContext(ctx, deleteSQL, id)
	if err != nil {
		return fmt.Errorf("delete memory: %w", err)
	}
	return nil
}

func (c *PGVectorClient) UpdateVector(ctx context.Context, id string, embedding []float32) error {
	vecStr := float32SliceToVector(embedding)
	updateSQL := fmt.Sprintf(
		`UPDATE %s SET embedding = $2::vector WHERE id = $1`,
		c.tableName,
	)
	_, err := c.db.ExecContext(ctx, updateSQL, id, vecStr)
	if err != nil {
		return fmt.Errorf("update vector: %w", err)
	}
	return nil
}

func (c *PGVectorClient) Ping(ctx context.Context) error {
	return c.db.PingContext(ctx)
}

func (c *PGVectorClient) Close() error {
	return c.db.Close()
}

func float32SliceToVector(vec []float32) string {
	parts := make([]string, len(vec))
	for i, v := range vec {
		parts[i] = fmt.Sprintf("%g", v)
		if math.IsNaN(float64(v)) {
			parts[i] = "0"
		}
	}
	return "[" + strings.Join(parts, ",") + "]"
}
