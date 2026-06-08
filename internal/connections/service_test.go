package connections

import (
	"context"
	"testing"

	"agent-memory/internal/sources"
)

type fakeSourceIngestor struct {
	ingested []sources.IngestRequest
	deleted  []string
}

func (f *fakeSourceIngestor) Ingest(ctx context.Context, req sources.IngestRequest) (*sources.IngestResult, error) {
	_ = ctx
	f.ingested = append(f.ingested, req)
	sourceID := "src_" + req.ExternalID
	if sourceID == "src_" {
		sourceID = "src_1"
	}
	return &sources.IngestResult{SourceID: sourceID, Status: "ingested", MemoriesCreated: 1}, nil
}

func (f *fakeSourceIngestor) Delete(ctx context.Context, sourceID string) error {
	_ = ctx
	f.deleted = append(f.deleted, sourceID)
	return nil
}

func TestCreateRedactsSecrets(t *testing.T) {
	svc := NewService(NewInMemoryStore(), &fakeSourceIngestor{})
	conn, err := svc.Create(context.Background(), ProviderNotion, CreateRequest{
		UserID: "user_1",
		Config: map[string]interface{}{
			"access_token": "secret",
			"workspace":    "engineering",
		},
	})
	if err != nil {
		t.Fatalf("create connection: %v", err)
	}
	if conn.Status != StatusActive {
		t.Fatalf("status = %s, want %s", conn.Status, StatusActive)
	}
	if conn.Config["access_token"] != "***redacted***" {
		t.Fatalf("access token was not redacted: %#v", conn.Config)
	}
	if conn.Config["workspace"] != "engineering" {
		t.Fatalf("non-secret config should be preserved: %#v", conn.Config)
	}
}

func TestSyncSeedDocumentsThroughSources(t *testing.T) {
	ingestor := &fakeSourceIngestor{}
	svc := NewService(NewInMemoryStore(), ingestor)
	conn, err := svc.Create(context.Background(), ProviderGitHub, CreateRequest{
		UserID: "user_1",
		Config: map[string]interface{}{
			"owner": "Himan-D",
			"repo":  "agent-memory",
		},
	})
	if err != nil {
		t.Fatalf("create connection: %v", err)
	}

	result, err := svc.Sync(context.Background(), conn.ID, SyncRequest{
		Documents: []SeedDocument{{
			Title:      "README",
			Content:    "Repository documentation",
			ExternalID: "readme",
			Metadata:   map[string]interface{}{"path": "README.md"},
		}},
	})
	if err != nil {
		t.Fatalf("sync connection: %v", err)
	}
	if result.Synced != 1 || len(result.SourceIDs) != 1 {
		t.Fatalf("sync result = %#v", result)
	}
	if len(ingestor.ingested) != 1 {
		t.Fatalf("ingested %d documents, want 1", len(ingestor.ingested))
	}
	req := ingestor.ingested[0]
	if req.Provider != ProviderGitHub || req.UserID != "user_1" || req.Metadata["connection_id"] != conn.ID {
		t.Fatalf("unexpected ingest request: %#v", req)
	}
}

func TestSyncMissingOAuthReportsRequired(t *testing.T) {
	svc := NewService(NewInMemoryStore(), &fakeSourceIngestor{})
	conn, err := svc.Create(context.Background(), ProviderSlack, CreateRequest{UserID: "user_1"})
	if err != nil {
		t.Fatalf("create connection: %v", err)
	}
	if conn.Status != StatusOAuthRequired {
		t.Fatalf("status = %s, want %s", conn.Status, StatusOAuthRequired)
	}
	result, err := svc.Sync(context.Background(), conn.ID, SyncRequest{})
	if err == nil {
		t.Fatal("expected sync error")
	}
	if result == nil || result.Status != StatusOAuthRequired {
		t.Fatalf("result = %#v, want oauth_required", result)
	}
}
