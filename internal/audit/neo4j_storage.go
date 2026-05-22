package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"agent-memory/internal/memory/neo4j"

	neo4jdriver "github.com/neo4j/neo4j-go-driver/v6/neo4j"
)

type Neo4jStorage struct {
	client *neo4j.Client
}

func NewNeo4jStorage(client *neo4j.Client) *Neo4jStorage {
	return &Neo4jStorage{client: client}
}

func (s *Neo4jStorage) Init(ctx context.Context) error {
	session, cleanup := s.client.GetSession(ctx)
	defer cleanup()

	indexes := []string{
		"CREATE INDEX audit_event_id_idx IF NOT EXISTS FOR (a:AuditEvent) ON (a.id)",
		"CREATE INDEX audit_event_tenant_idx IF NOT EXISTS FOR (a:AuditEvent) ON (a.tenant_id)",
		"CREATE INDEX audit_event_timestamp_idx IF NOT EXISTS FOR (a:AuditEvent) ON (a.timestamp)",
		"CREATE INDEX audit_event_type_idx IF NOT EXISTS FOR (a:AuditEvent) ON (a.type)",
		"CREATE INDEX audit_event_actor_idx IF NOT EXISTS FOR (a:AuditEvent) ON (a.actor_id)",
	}

	for _, idx := range indexes {
		_, err := session.Run(ctx, idx, nil)
		if err != nil {
			return fmt.Errorf("audit init index: %w", err)
		}
	}
	return nil
}

func (s *Neo4jStorage) Store(ctx context.Context, event *Event) error {
	session, cleanup := s.client.GetSession(ctx)
	defer cleanup()

	var metadataJSON string
	if event.Metadata != nil {
		data, err := json.Marshal(event.Metadata)
		if err != nil {
			return fmt.Errorf("audit store marshal metadata: %w", err)
		}
		metadataJSON = string(data)
	}

	query := `
		CREATE (a:AuditEvent {
			id: $id,
			tenant_id: $tenant_id,
			timestamp: datetime($timestamp),
			type: $type,
			actor_id: $actor_id,
			actor_type: $actor_type,
			resource_type: $resource_type,
			resource_id: $resource_id,
			action: $action,
			status: $status,
			ip_address: $ip_address,
			user_agent: $user_agent,
			metadata_json: $metadata_json,
			error: $error,
			duration_ms: $duration_ms
		})
		RETURN a.id
	`

	_, err := session.Run(ctx, query, map[string]interface{}{
		"id":            event.ID,
		"tenant_id":     event.TenantID,
		"timestamp":     event.Timestamp.Format(time.RFC3339),
		"type":          string(event.Type),
		"actor_id":      event.ActorID,
		"actor_type":    event.ActorType,
		"resource_type": event.ResourceType,
		"resource_id":   event.ResourceID,
		"action":        event.Action,
		"status":        event.Status,
		"ip_address":    event.IPAddress,
		"user_agent":    event.UserAgent,
		"metadata_json": metadataJSON,
		"error":         event.Error,
		"duration_ms":   event.DurationMs,
	})
	if err != nil {
		return fmt.Errorf("audit store: %w", err)
	}
	return nil
}

func (s *Neo4jStorage) Query(ctx context.Context, filter *Filter) ([]*Event, error) {
	session, cleanup := s.client.GetSession(ctx)
	defer cleanup()

	where, params := s.buildFilterConditions(filter)

	query := fmt.Sprintf(`
		MATCH (a:AuditEvent)
		WHERE %s
		RETURN a.id, a.tenant_id, a.timestamp, a.type, a.actor_id, a.actor_type,
		       a.resource_type, a.resource_id, a.action, a.status,
		       a.ip_address, a.user_agent, a.metadata_json, a.error, a.duration_ms
		ORDER BY a.timestamp DESC
	`, where)

	if filter != nil {
		if filter.Limit > 0 {
			query += fmt.Sprintf(" LIMIT $limit")
			params["limit"] = int64(filter.Limit)
		}
		if filter.Offset > 0 {
			query += fmt.Sprintf(" SKIP $offset")
			params["offset"] = int64(filter.Offset)
		}
	}

	result, err := session.Run(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("audit query: %w", err)
	}

	var events []*Event
	for result.Next(ctx) {
		event, err := recordToEvent(result.Record())
		if err != nil {
			continue
		}
		events = append(events, event)
	}
	return events, nil
}

func (s *Neo4jStorage) Count(ctx context.Context, filter *Filter) (int, error) {
	session, cleanup := s.client.GetSession(ctx)
	defer cleanup()

	where, params := s.buildFilterConditions(filter)

	query := fmt.Sprintf(`
		MATCH (a:AuditEvent)
		WHERE %s
		RETURN count(a) AS cnt
	`, where)

	result, err := session.Run(ctx, query, params)
	if err != nil {
		return 0, fmt.Errorf("audit count: %w", err)
	}

	if result.Next(ctx) {
		rec := result.Record()
		if cnt, ok := rec.Values[0].(int64); ok {
			return int(cnt), nil
		}
	}
	return 0, nil
}

func (s *Neo4jStorage) DeleteOld(ctx context.Context, before time.Time) (int, error) {
	session, cleanup := s.client.GetSession(ctx)
	defer cleanup()

	query := `
		MATCH (a:AuditEvent)
		WHERE a.timestamp < datetime($before)
		DELETE a
		RETURN count(a) AS deleted
	`

	result, err := session.Run(ctx, query, map[string]interface{}{
		"before": before.Format(time.RFC3339),
	})
	if err != nil {
		return 0, fmt.Errorf("audit delete old: %w", err)
	}

	if result.Next(ctx) {
		rec := result.Record()
		if cnt, ok := rec.Values[0].(int64); ok {
			return int(cnt), nil
		}
	}
	return 0, nil
}

func (s *Neo4jStorage) buildFilterConditions(filter *Filter) (string, map[string]interface{}) {
	if filter == nil {
		return "true", map[string]interface{}{}
	}

	conditions := []string{}
	params := map[string]interface{}{}
	paramIdx := 0

	nextParam := func(name string) string {
		paramIdx++
		full := fmt.Sprintf("%s_%d", name, paramIdx)
		return full
	}

	if filter.TenantID != "" {
		p := nextParam("tenant_id")
		conditions = append(conditions, fmt.Sprintf("a.tenant_id = $%s", p))
		params[p] = filter.TenantID
	}

	if !filter.StartTime.IsZero() {
		p := nextParam("start_time")
		conditions = append(conditions, fmt.Sprintf("a.timestamp >= datetime($%s)", p))
		params[p] = filter.StartTime.Format(time.RFC3339)
	}

	if !filter.EndTime.IsZero() {
		p := nextParam("end_time")
		conditions = append(conditions, fmt.Sprintf("a.timestamp <= datetime($%s)", p))
		params[p] = filter.EndTime.Format(time.RFC3339)
	}

	if len(filter.Types) > 0 {
		p := nextParam("types")
		typeStrings := make([]string, len(filter.Types))
		for i, t := range filter.Types {
			typeStrings[i] = string(t)
		}
		conditions = append(conditions, fmt.Sprintf("a.type IN $%s", p))
		params[p] = typeStrings
	}

	if filter.ActorID != "" {
		p := nextParam("actor_id")
		conditions = append(conditions, fmt.Sprintf("a.actor_id = $%s", p))
		params[p] = filter.ActorID
	}

	if filter.ResourceType != "" {
		p := nextParam("resource_type")
		conditions = append(conditions, fmt.Sprintf("a.resource_type = $%s", p))
		params[p] = filter.ResourceType
	}

	if filter.ResourceID != "" {
		p := nextParam("resource_id")
		conditions = append(conditions, fmt.Sprintf("a.resource_id = $%s", p))
		params[p] = filter.ResourceID
	}

	if filter.Status != "" {
		p := nextParam("status")
		conditions = append(conditions, fmt.Sprintf("a.status = $%s", p))
		params[p] = filter.Status
	}

	if len(conditions) == 0 {
		return "true", params
	}

	return strings.Join(conditions, " AND "), params
}

func recordToEvent(rec *neo4jdriver.Record) (*Event, error) {
	vals := rec.Values

	event := &Event{}
	if len(vals) > 0 {
		event.ID = asString(vals[0])
	}
	if len(vals) > 1 {
		event.TenantID = asString(vals[1])
	}
	if len(vals) > 2 && vals[2] != nil {
		if t, ok := vals[2].(time.Time); ok {
			event.Timestamp = t
		}
	}
	if len(vals) > 3 {
		event.Type = EventType(asString(vals[3]))
	}
	if len(vals) > 4 {
		event.ActorID = asString(vals[4])
	}
	if len(vals) > 5 {
		event.ActorType = asString(vals[5])
	}
	if len(vals) > 6 {
		event.ResourceType = asString(vals[6])
	}
	if len(vals) > 7 {
		event.ResourceID = asString(vals[7])
	}
	if len(vals) > 8 {
		event.Action = asString(vals[8])
	}
	if len(vals) > 9 {
		event.Status = asString(vals[9])
	}
	if len(vals) > 10 {
		event.IPAddress = asString(vals[10])
	}
	if len(vals) > 11 {
		event.UserAgent = asString(vals[11])
	}
	if len(vals) > 12 && vals[12] != nil {
		if metaStr, ok := vals[12].(string); ok && metaStr != "" {
			event.Metadata = make(map[string]interface{})
			_ = json.Unmarshal([]byte(metaStr), &event.Metadata)
		}
	}
	if len(vals) > 13 {
		event.Error = asString(vals[13])
	}
	if len(vals) > 14 && vals[14] != nil {
		switch v := vals[14].(type) {
		case int64:
			event.DurationMs = v
		case float64:
			event.DurationMs = int64(v)
		}
	}

	return event, nil
}

func asString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}
