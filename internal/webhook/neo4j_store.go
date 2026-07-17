package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	neo4jdriver "github.com/neo4j/neo4j-go-driver/v6/neo4j"

	"agent-memory/internal/memory/neo4j"
	"agent-memory/internal/memory/types"
)

type Neo4jStore struct {
	client *neo4j.Client
}

func NewNeo4jStore(client *neo4j.Client) *Neo4jStore {
	return &Neo4jStore{client: client}
}

func (s *Neo4jStore) Init(ctx context.Context) error {
	session, cleanup, err := s.client.AcquireSession(ctx)
	if err != nil {
		return fmt.Errorf("webhook store init: %w", err)
	}
	defer cleanup()

	indexes := []string{
		"CREATE INDEX webhook_id_idx IF NOT EXISTS FOR (w:Webhook) ON (w.id)",
		"CREATE INDEX webhook_project_id_idx IF NOT EXISTS FOR (w:Webhook) ON (w.project_id)",
		"CREATE CONSTRAINT webhook_id_unique IF NOT EXISTS FOR (w:Webhook) REQUIRE w.id IS UNIQUE",
		"CREATE INDEX webhook_dlq_id_idx IF NOT EXISTS FOR (d:WebhookDeadLetter) ON (d.id)",
		"CREATE INDEX webhook_dlq_webhook_id_idx IF NOT EXISTS FOR (d:WebhookDeadLetter) ON (d.webhook_id)",
		"CREATE INDEX webhook_delivery_id_idx IF NOT EXISTS FOR (d:WebhookDelivery) ON (d.id)",
	}

	for _, idx := range indexes {
		_, err := session.Run(ctx, idx, nil)
		if err != nil {
			return fmt.Errorf("webhook store init index: %w", err)
		}
	}
	return nil
}

func (s *Neo4jStore) Store(ctx context.Context, wh *types.Webhook) error {
	eventsJSON, err := json.Marshal(wh.Events)
	if err != nil {
		return fmt.Errorf("webhook store marshal events: %w", err)
	}
	fieldsJSON, err := json.Marshal(wh.Fields)
	if err != nil {
		return fmt.Errorf("webhook store marshal fields: %w", err)
	}

	metadataJSON, err := json.Marshal(wh.Metadata)
	if err != nil {
		return fmt.Errorf("webhook store marshal metadata: %w", err)
	}

	session, cleanup, err := s.client.AcquireSession(ctx)
	if err != nil {
		return fmt.Errorf("webhook store acquire session: %w", err)
	}
	defer cleanup()

	query := `
		MERGE (w:Webhook {id: $id})
		SET w.project_id = $project_id,
		    w.tenant_id = $tenant_id,
		    w.url = $url,
		    w.events_json = $events_json,
		    w.fields_json = $fields_json,
		    w.secret = $secret,
		    w.active = $active,
		    w.metadata_json = $metadata_json,
		    w.created_at = datetime($created_at),
		    w.verified_at = $verified_at,
		    w.success_count = $success_count,
		    w.failure_count = $failure_count,
		    w.last_status_code = $last_status_code,
		    w.last_delivery_at = $last_delivery_at
		RETURN w.id
	`

	result, err := session.Run(ctx, query, map[string]interface{}{
		"id":               wh.ID,
		"project_id":       wh.ProjectID,
		"tenant_id":        wh.TenantID,
		"url":              wh.URL,
		"events_json":      string(eventsJSON),
		"fields_json":      string(fieldsJSON),
		"secret":           wh.Secret,
		"active":           wh.Active,
		"metadata_json":    string(metadataJSON),
		"created_at":       wh.CreatedAt.Format(time.RFC3339),
		"verified_at":      nilIfNilTime(wh.VerifiedAt),
		"success_count":    wh.SuccessCount,
		"failure_count":    wh.FailureCount,
		"last_status_code": wh.LastStatusCode,
		"last_delivery_at": nilIfNilTime(wh.LastDeliveryAt),
	})
	if err != nil {
		return fmt.Errorf("webhook store: %w", err)
	}

	_, err = result.Consume(ctx)
	return err
}

func (s *Neo4jStore) Get(ctx context.Context, id string) (*types.Webhook, error) {
	session, cleanup, err := s.client.AcquireSession(ctx)
	if err != nil {
		return nil, fmt.Errorf("webhook get acquire session: %w", err)
	}
	defer cleanup()

	query := `
		MATCH (w:Webhook {id: $id})
		RETURN w.id, w.project_id, w.tenant_id, w.url, w.events_json, w.fields_json, w.secret, w.active, w.metadata_json, w.created_at, w.verified_at
	`

	result, err := session.Run(ctx, query, map[string]interface{}{"id": id})
	if err != nil {
		return nil, fmt.Errorf("webhook get: %w", err)
	}

	if result.Next(ctx) {
		return s.recordToWebhook(result.Record())
	}
	return nil, fmt.Errorf("webhook not found: %s", id)
}

func (s *Neo4jStore) Update(ctx context.Context, wh *types.Webhook) error {
	eventsJSON, err := json.Marshal(wh.Events)
	if err != nil {
		return fmt.Errorf("webhook update marshal events: %w", err)
	}
	fieldsJSON, err := json.Marshal(wh.Fields)
	if err != nil {
		return fmt.Errorf("webhook update marshal fields: %w", err)
	}

	metadataJSON, err := json.Marshal(wh.Metadata)
	if err != nil {
		return fmt.Errorf("webhook update marshal metadata: %w", err)
	}

	session, cleanup, err := s.client.AcquireSession(ctx)
	if err != nil {
		return fmt.Errorf("webhook update acquire session: %w", err)
	}
	defer cleanup()

	query := `
		MATCH (w:Webhook {id: $id})
		SET w.project_id = $project_id,
		    w.tenant_id = $tenant_id,
		    w.url = $url,
		    w.events_json = $events_json,
		    w.fields_json = $fields_json,
		    w.secret = $secret,
		    w.active = $active,
		    w.metadata_json = $metadata_json,
		    w.verified_at = $verified_at,
		    w.success_count = $success_count,
		    w.failure_count = $failure_count,
		    w.last_status_code = $last_status_code,
		    w.last_delivery_at = $last_delivery_at
		RETURN w.id
	`

	result, err := session.Run(ctx, query, map[string]interface{}{
		"id":               wh.ID,
		"project_id":       wh.ProjectID,
		"tenant_id":        wh.TenantID,
		"url":              wh.URL,
		"events_json":      string(eventsJSON),
		"fields_json":      string(fieldsJSON),
		"secret":           wh.Secret,
		"active":           wh.Active,
		"metadata_json":    string(metadataJSON),
		"verified_at":      nilIfNilTime(wh.VerifiedAt),
		"success_count":    wh.SuccessCount,
		"failure_count":    wh.FailureCount,
		"last_status_code": wh.LastStatusCode,
		"last_delivery_at": nilIfNilTime(wh.LastDeliveryAt),
	})
	if err != nil {
		return fmt.Errorf("webhook update: %w", err)
	}

	if !result.Next(ctx) {
		return fmt.Errorf("webhook not found for update: %s", wh.ID)
	}
	_, err = result.Consume(ctx)
	return err
}

func (s *Neo4jStore) Delete(ctx context.Context, id string) error {
	session, cleanup, err := s.client.AcquireSession(ctx)
	if err != nil {
		return fmt.Errorf("webhook delete acquire session: %w", err)
	}
	defer cleanup()

	query := `
		MATCH (w:Webhook {id: $id})
		DETACH DELETE w
	`

	result, err := session.Run(ctx, query, map[string]interface{}{"id": id})
	if err != nil {
		return fmt.Errorf("webhook delete: %w", err)
	}
	_, err = result.Consume(ctx)
	return err
}

func (s *Neo4jStore) List(ctx context.Context, projectID string) ([]*types.Webhook, error) {
	session, cleanup, err := s.client.AcquireSession(ctx)
	if err != nil {
		return nil, fmt.Errorf("webhook list acquire session: %w", err)
	}
	defer cleanup()

	var query string
	var params map[string]interface{}

	if projectID != "" {
		query = `
			MATCH (w:Webhook {project_id: $project_id})
			RETURN w.id, w.project_id, w.tenant_id, w.url, w.events_json, w.fields_json, w.secret, w.active, w.metadata_json, w.created_at, w.verified_at
			ORDER BY w.created_at DESC
		`
		params = map[string]interface{}{"project_id": projectID}
	} else {
		query = `
			MATCH (w:Webhook)
			RETURN w.id, w.project_id, w.tenant_id, w.url, w.events_json, w.fields_json, w.secret, w.active, w.metadata_json, w.created_at, w.verified_at
			ORDER BY w.created_at DESC
		`
		params = map[string]interface{}{}
	}

	result, err := session.Run(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("webhook list: %w", err)
	}

	var webhooks []*types.Webhook
	for result.Next(ctx) {
		wh, err := s.recordToWebhook(result.Record())
		if err != nil {
			continue
		}
		webhooks = append(webhooks, wh)
	}
	return webhooks, nil
}

func (s *Neo4jStore) ListActive(ctx context.Context, projectID string) ([]*types.Webhook, error) {
	session, cleanup, err := s.client.AcquireSession(ctx)
	if err != nil {
		return nil, fmt.Errorf("webhook list active acquire session: %w", err)
	}
	defer cleanup()

	var query string
	var params map[string]interface{}

	if projectID != "" {
		query = `
			MATCH (w:Webhook {active: true, project_id: $project_id})
			RETURN w.id, w.project_id, w.tenant_id, w.url, w.events_json, w.fields_json, w.secret, w.active, w.metadata_json, w.created_at, w.verified_at
			ORDER BY w.created_at DESC
		`
		params = map[string]interface{}{"project_id": projectID}
	} else {
		query = `
			MATCH (w:Webhook {active: true})
			RETURN w.id, w.project_id, w.tenant_id, w.url, w.events_json, w.fields_json, w.secret, w.active, w.metadata_json, w.created_at, w.verified_at
			ORDER BY w.created_at DESC
		`
		params = map[string]interface{}{}
	}

	result, err := session.Run(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("webhook list active: %w", err)
	}

	var webhooks []*types.Webhook
	for result.Next(ctx) {
		wh, err := s.recordToWebhook(result.Record())
		if err != nil {
			continue
		}
		webhooks = append(webhooks, wh)
	}
	return webhooks, nil
}

func (s *Neo4jStore) recordToWebhook(rec *neo4jdriver.Record) (*types.Webhook, error) {
	getString := func(idx int) string {
		if idx < len(rec.Values) && rec.Values[idx] != nil {
			if v, ok := rec.Values[idx].(string); ok {
				return v
			}
		}
		return ""
	}

	getBool := func(idx int) bool {
		if idx < len(rec.Values) && rec.Values[idx] != nil {
			if v, ok := rec.Values[idx].(bool); ok {
				return v
			}
		}
		return false
	}

	wh := &types.Webhook{
		ID:        getString(0),
		ProjectID: getString(1),
		TenantID:  getString(2),
		URL:       getString(3),
		Secret:    getString(6),
		Active:    getBool(7),
	}

	eventsStr := getString(4)
	if eventsStr != "" {
		var events []types.WebhookEvent
		if err := json.Unmarshal([]byte(eventsStr), &events); err != nil {
			return nil, fmt.Errorf("webhook unmarshal events: %w", err)
		}
		wh.Events = events
	}

	fieldsStr := getString(5)
	if fieldsStr != "" {
		var fields []string
		if err := json.Unmarshal([]byte(fieldsStr), &fields); err != nil {
			return nil, fmt.Errorf("webhook unmarshal fields: %w", err)
		}
		wh.Fields = fields
	}

	metadataStr := getString(8)
	if metadataStr != "" {
		var metadata map[string]interface{}
		if err := json.Unmarshal([]byte(metadataStr), &metadata); err != nil {
			return nil, fmt.Errorf("webhook unmarshal metadata: %w", err)
		}
		wh.Metadata = metadata
	}

	if len(rec.Values) > 9 && rec.Values[9] != nil {
		if t, ok := rec.Values[9].(time.Time); ok {
			wh.CreatedAt = t
		}
	}
	if len(rec.Values) > 10 && rec.Values[10] != nil {
		if t, ok := rec.Values[10].(time.Time); ok {
			wh.VerifiedAt = &t
		}
	}

	return wh, nil
}

func nilIfNilTime(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return t.Format(time.RFC3339)
}

// StoreDeadLetter persists a dead-letter entry so it survives process restarts
// even when the in-memory/file cache is cleared.
func (s *Neo4jStore) StoreDeadLetter(ctx context.Context, entry DeadLetterEntry) error {
	session, cleanup, err := s.client.AcquireSession(ctx)
	if err != nil {
		return fmt.Errorf("webhook dlq acquire: %w", err)
	}
	defer cleanup()

	payloadJSON, _ := json.Marshal(entry.Payload)
	query := `
		MERGE (d:WebhookDeadLetter {id: $id})
		SET d.webhook_id = $webhook_id,
		    d.event = $event,
		    d.error = $error,
		    d.attempts = $attempts,
		    d.payload_json = $payload_json,
		    d.failed_at = datetime($failed_at),
		    d.created_at = datetime($created_at)
		WITH d
		OPTIONAL MATCH (w:Webhook {id: $webhook_id})
		FOREACH (_ IN CASE WHEN w IS NULL THEN [] ELSE [1] END |
			MERGE (w)-[:HAS_DEAD_LETTER]->(d)
		)
		RETURN d.id
	`
	result, err := session.Run(ctx, query, map[string]interface{}{
		"id":           entry.ID,
		"webhook_id":   entry.WebhookID,
		"event":        string(entry.Event),
		"error":        entry.Error,
		"attempts":     entry.Attempts,
		"payload_json": string(payloadJSON),
		"failed_at":    entry.FailedAt.Format(time.RFC3339),
		"created_at":   entry.CreatedAt.Format(time.RFC3339),
	})
	if err != nil {
		return fmt.Errorf("webhook dlq store: %w", err)
	}
	_, err = result.Consume(ctx)
	return err
}

// StoreDelivery persists a delivery log entry for audit/history.
func (s *Neo4jStore) StoreDelivery(ctx context.Context, log DeliveryLog) error {
	session, cleanup, err := s.client.AcquireSession(ctx)
	if err != nil {
		return fmt.Errorf("webhook delivery acquire: %w", err)
	}
	defer cleanup()

	query := `
		MERGE (d:WebhookDelivery {id: $id})
		SET d.webhook_id = $webhook_id,
		    d.event = $event,
		    d.attempt = $attempt,
		    d.success = $success,
		    d.status = $status,
		    d.status_code = $status_code,
		    d.error = $error,
		    d.duration_ms = $duration_ms,
		    d.created_at = datetime($created_at)
		WITH d
		OPTIONAL MATCH (w:Webhook {id: $webhook_id})
		FOREACH (_ IN CASE WHEN w IS NULL THEN [] ELSE [1] END |
			MERGE (w)-[:HAS_DELIVERY]->(d)
		)
		RETURN d.id
	`
	result, err := session.Run(ctx, query, map[string]interface{}{
		"id":           log.ID,
		"webhook_id":   log.WebhookID,
		"event":        log.Event,
		"attempt":      log.Attempt,
		"success":      log.Success,
		"status":       log.Status,
		"status_code":  log.StatusCode,
		"error":        log.Error,
		"duration_ms":  log.DurationMs,
		"created_at":   log.CreatedAt.Format(time.RFC3339),
	})
	if err != nil {
		return fmt.Errorf("webhook delivery store: %w", err)
	}
	_, err = result.Consume(ctx)
	return err
}
