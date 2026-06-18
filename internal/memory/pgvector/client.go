package pgvector

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"

	"agent-memory/internal/config"
	"agent-memory/internal/memory/types"
)

var tableNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

// Client implements VectorStore using PostgreSQL with the pgvector extension.
type Client struct {
	db  *sql.DB
	cfg config.PgvectorConfig
}

func NewClient(cfg config.PgvectorConfig) (*Client, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("pgvector: PGVECTOR_URL is required")
	}

	// Validate the table name to prevent SQL injection since it's used in format strings.
	// We allow only alphanumeric characters and underscores.
	// If a user needs schema-qualified tables like "public.my_vectors", they must adapt this.
	if !tableNameRegex.MatchString(cfg.Table) {
		return nil, fmt.Errorf("pgvector: invalid table name format: %q", cfg.Table)
	}

	db, err := sql.Open("postgres", cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("pgvector: open db: %w", err)
	}

	c := &Client{db: db, cfg: cfg}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := c.ensureSchema(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("pgvector: ensure schema: %w", err)
	}

	return c, nil
}

func (c *Client) Close() error {
	return c.db.Close()
}

func (c *Client) Ping(ctx context.Context) error {
	return c.db.PingContext(ctx)
}

func (c *Client) ensureSchema(ctx context.Context) error {
	_, err := c.db.ExecContext(ctx, `CREATE EXTENSION IF NOT EXISTS vector`)
	if err != nil {
		return fmt.Errorf("create extension vector: %w", err)
	}

	_, err = c.db.ExecContext(ctx, fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id          TEXT PRIMARY KEY,
			text        TEXT NOT NULL,
			entity_id   TEXT,
			embedding   vector(%d),
			metadata    JSONB,
			created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
		)`, c.cfg.Table, c.cfg.Dimensions))
	if err != nil {
		return fmt.Errorf("create table: %w", err)
	}

	// Index for approximate nearest-neighbour search (cosine distance).
	_, err = c.db.ExecContext(ctx, fmt.Sprintf(`
		CREATE INDEX IF NOT EXISTS %s_embedding_cosine_idx
		ON %s USING ivfflat (embedding vector_cosine_ops)
		WITH (lists = 100)`, c.cfg.Table, c.cfg.Table))
	if err != nil {
		// ivfflat requires data; ignore the error if the table is empty at setup time.
		// The index will need to be created manually once data is present.
		_ = err
	}

	// Index on entity_id for filtered lookups.
	_, err = c.db.ExecContext(ctx, fmt.Sprintf(`
		CREATE INDEX IF NOT EXISTS %s_entity_id_idx ON %s (entity_id)`,
		c.cfg.Table, c.cfg.Table))
	if err != nil {
		return fmt.Errorf("create entity_id index: %w", err)
	}

	return nil
}

func (c *Client) StoreEmbedding(
	ctx context.Context,
	text string,
	id string,
	embedding []float32,
	metadata map[string]interface{},
) (string, error) {
	pointID := uuid.New().String()

	if metadata == nil {
		metadata = map[string]interface{}{}
	}
	metadata["entity_id"] = id

	metaJSON, err := json.Marshal(metadata)
	if err != nil {
		return "", fmt.Errorf("pgvector store: marshal metadata: %w", err)
	}

	vec := float32SliceToLiteral(embedding)

	query := fmt.Sprintf(`
		INSERT INTO %s (id, text, entity_id, embedding, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4::vector, $5, $6, $7)`,
		c.cfg.Table)

	now := time.Now().UTC()
	_, err = c.db.ExecContext(ctx, query,
		pointID, text, id, vec, metaJSON, now, now)
	if err != nil {
		return "", fmt.Errorf("pgvector store: insert: %w", err)
	}
	return pointID, nil
}

func (c *Client) Search(
	ctx context.Context,
	query []float32,
	limit int,
	threshold float32,
	filters map[string]interface{},
) ([]types.MemoryResult, error) {
	vec := float32SliceToLiteral(query)

	var whereClauses []string
	var args []interface{}
	// First arg is the query vector.
	args = append(args, vec)

	argIdx := 2
	for k, v := range filters {
		whereClauses = append(whereClauses,
			fmt.Sprintf("metadata->>'%s' = $%d", sanitizeKey(k), argIdx))
		args = append(args, fmt.Sprintf("%v", v))
		argIdx++
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	// cosine similarity = 1 - cosine_distance
	sqlStr := fmt.Sprintf(`
		SELECT id, text, entity_id, metadata,
		       1 - (embedding <=> $1::vector) AS score
		FROM %s
		%s
		ORDER BY embedding <=> $1::vector
		LIMIT %d`, c.cfg.Table, whereSQL, limit)

	rows, err := c.db.QueryContext(ctx, sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("pgvector search: query: %w", err)
	}
	defer rows.Close()

	var results []types.MemoryResult
	for rows.Next() {
		var (
			pid      string
			text     string
			entityID sql.NullString
			metaRaw  []byte
			score    float32
		)
		if err := rows.Scan(&pid, &text, &entityID, &metaRaw, &score); err != nil {
			return nil, fmt.Errorf("pgvector search: scan: %w", err)
		}
		if score < threshold {
			continue
		}

		meta := map[string]interface{}{}
		if len(metaRaw) > 0 {
			_ = json.Unmarshal(metaRaw, &meta)
		}

		eid := ""
		if entityID.Valid {
			eid = entityID.String
		}

		results = append(results, types.MemoryResult{
			Entity: types.Entity{
				ID:         eid,
				Properties: meta,
			},
			Score:    score,
			Text:     text,
			Source:   "pgvector",
			MemoryID: pid,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("pgvector search: rows: %w", err)
	}
	return results, nil
}

func (c *Client) UpdateMemory(
	ctx context.Context,
	id string,
	text string,
	metadata map[string]interface{},
) error {
	metaJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("pgvector update: marshal metadata: %w", err)
	}

	query := fmt.Sprintf(`
		UPDATE %s SET text = $1, metadata = $2, updated_at = $3
		WHERE id = $4`, c.cfg.Table)

	_, err = c.db.ExecContext(ctx, query, text, metaJSON, time.Now().UTC(), id)
	if err != nil {
		return fmt.Errorf("pgvector update memory: %w", err)
	}
	return nil
}

func (c *Client) DeleteMemory(ctx context.Context, id string) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE id = $1`, c.cfg.Table)
	_, err := c.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("pgvector delete memory: %w", err)
	}
	return nil
}

func (c *Client) UpdateVector(ctx context.Context, id string, embedding []float32) error {
	vec := float32SliceToLiteral(embedding)
	query := fmt.Sprintf(`
		UPDATE %s SET embedding = $1::vector, updated_at = $2
		WHERE id = $3`, c.cfg.Table)

	_, err := c.db.ExecContext(ctx, query, vec, time.Now().UTC(), id)
	if err != nil {
		return fmt.Errorf("pgvector update vector: %w", err)
	}
	return nil
}

// float32SliceToLiteral converts a float32 slice to the pgvector literal format: '[1.0,2.0,...]'.
func float32SliceToLiteral(v []float32) string {
	sb := strings.Builder{}
	sb.WriteByte('[')
	for i, f := range v {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(fmt.Sprintf("%g", f))
	}
	sb.WriteByte(']')
	return sb.String()
}

// sanitizeKey removes characters that could be used for SQL injection in metadata key names.
// Since keys come from application code, this is a defence-in-depth measure.
func sanitizeKey(k string) string {
	var sb strings.Builder
	for _, r := range k {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '_' {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}
