package improve

import (
	"context"
	"fmt"
	"time"
)

// cacheClient is the minimal Redis surface used by SyncToCache.
type cacheClient interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
}

// SyncToCache pushes the latest GlobalContext from Neo4j into Redis so the
// next session turn starts with richer context. This is stage 6 of the
// improvement pipeline and the natural complement to GlobalContextIndex.
//
// Redis key schema:
//   global_context:{tenant_id}              -> JSON or string payload
//   global_context:{tenant_id}:{dataset_id} -> JSON or string payload
//
// The TTL is configurable via CacheTTL (zero = 24h).
type SyncToCache struct {
	Graph    graphExec
	Cache    cacheClient
	TenantID string
	DatasetID string
	CacheTTL time.Duration
}

// Name implements Stage.
func (s SyncToCache) Name() string { return "sync_to_cache" }

// Run implements Stage.
func (s SyncToCache) Run(ctx context.Context, in Input) (StageResult, error) {
	start := time.Now().UTC()
	res := StageResult{Name: s.Name(), Started: start}

	if s.Graph == nil {
		res.Ended = time.Now().UTC()
		return res, fmt.Errorf("sync_to_cache: graph executor is nil")
	}
	if s.Cache == nil {
		res.Ended = time.Now().UTC()
		return res, fmt.Errorf("sync_to_cache: cache client is nil")
	}

	tenant := s.TenantID
	if tenant == "" {
		tenant = in.UserID
	}
	if tenant == "" {
		res.Ended = time.Now().UTC()
		return res, fmt.Errorf("sync_to_cache: tenant_id is required")
	}

	id := "gc:" + tenant
	if s.DatasetID != "" {
		id = "gc:" + tenant + ":" + s.DatasetID
	}

	rows, err := s.Graph.QueryGraph(`
MATCH (g:GlobalContext {id: $id})
RETURN g.summary AS summary, g.entity_glossary AS glossary,
       g.tenant_id AS tenant_id, g.dataset_id AS dataset_id,
       g.updated_at AS updated_at
`, map[string]interface{}{"id": id})
	if err != nil {
		res.Ended = time.Now().UTC()
		return res, fmt.Errorf("sync_to_cache: query global_context: %w", err)
	}
	if len(rows) == 0 {
		// No global context yet; nothing to sync. This is not an error.
		res.Items = 0
		res.Ended = time.Now().UTC()
		return res, nil
	}

	r := rows[0]
	summary := asString(r["summary"])
	glossary := asStringSlice(r["entity_glossary"])

	ttl := s.CacheTTL
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}

	key := "global_context:" + tenant
	if s.DatasetID != "" {
		key = "global_context:" + tenant + ":" + s.DatasetID
	}

	// Payload is a compact newline-separated representation that callers
	// can parse trivially without depending on a JSON schema.
	payload := summary
	if len(glossary) > 0 {
		payload = summary + "\n---\nentities: " + joinStrings(glossary, ",")
	}
	if err := s.Cache.Set(ctx, key, payload, ttl); err != nil {
		res.Ended = time.Now().UTC()
		return res, fmt.Errorf("sync_to_cache: set %s: %w", key, err)
	}

	res.Items = 1
	res.Ended = time.Now().UTC()
	return res, nil
}

// asStringSlice converts a []interface{} (from a Cypher list param) into a
// []string. Returns nil for nil input.
func asStringSlice(v interface{}) []string {
	if v == nil {
		return nil
	}
	list, ok := v.([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// joinStrings joins s with sep. Returns "" for nil.
func joinStrings(s []string, sep string) string {
	if len(s) == 0 {
		return ""
	}
	out := s[0]
	for _, v := range s[1:] {
		out += sep + v
	}
	return out
}
