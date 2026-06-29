package cogni

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"

	"agent-memory/internal/memory/improve"
	"agent-memory/internal/memory/rollback"
	"agent-memory/internal/memory/types"
	"agent-memory/internal/retrieval"
	"agent-memory/internal/session"
)

func newTestRouter(t *testing.T, deps Deps) *mux.Router {
	t.Helper()
	r := mux.NewRouter()
	RegisterRoutes(r, deps)
	return r
}

func TestAddQA_HappyPath(t *testing.T) {
	mgr := session.NewManager(session.NewInMemoryStore())
	r := newTestRouter(t, Deps{SessionManager: mgr})

	body, _ := json.Marshal(addQARequest{
		UserID:   "u1",
		Question: "Q?",
		Answer:   "A.",
	})
	req := httptest.NewRequest(http.MethodPost, "/sessions/s1/qa", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp addQAResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Turn.Question != "Q?" || resp.Turn.Answer != "A." {
		t.Fatalf("unexpected turn: %+v", resp.Turn)
	}
}

func TestAddQA_MissingUserID(t *testing.T) {
	mgr := session.NewManager(session.NewInMemoryStore())
	r := newTestRouter(t, Deps{SessionManager: mgr})

	body, _ := json.Marshal(addQARequest{Question: "Q?"})
	req := httptest.NewRequest(http.MethodPost, "/sessions/s1/qa", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestListQA_RequiresUserIDQueryParam(t *testing.T) {
	mgr := session.NewManager(session.NewInMemoryStore())
	r := newTestRouter(t, Deps{SessionManager: mgr})

	req := httptest.NewRequest(http.MethodGet, "/sessions/s1/qa", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestImproveSession_503WhenNotConfigured(t *testing.T) {
	r := newTestRouter(t, Deps{})
	body := strings.NewReader(`{"user_id":"u1"}`)
	req := httptest.NewRequest(http.MethodPost, "/sessions/s1/improve", body)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestImproveSession_RunsPipeline(t *testing.T) {
	r := newTestRouter(t, Deps{Improver: improve.NewPipeline()})
	body := strings.NewReader(`{"user_id":"u1","build_global_context":true,"run_sync_to_cache":true}`)
	req := httptest.NewRequest(http.MethodPost, "/sessions/s1/improve", body)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var out improve.Output
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if _, ok := out.Stages["global_context_index"]; !ok {
		t.Fatal("expected global_context_index stage to have run")
	}
}

func TestDistillSession_503WhenNotConfigured(t *testing.T) {
	r := newTestRouter(t, Deps{})
	body := strings.NewReader(`{"user_id":"u1"}`)
	req := httptest.NewRequest(http.MethodPost, "/sessions/s1/distill", body)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestDistillSession_RunsDistiller(t *testing.T) {
	mgr := session.NewManager(session.NewInMemoryStore())
	_, _ = mgr.AddQATurn(context.Background(), "u1", "s1", "Q?", "A.", "", nil, nil)
	d := session.NewDistiller(mgr, session.DistillOptions{
		Curator: func(ctx context.Context, _ string) ([]session.ProposedLesson, error) {
			return []session.ProposedLesson{{WorkingStatement: "test"}}, nil
		},
		Writer: session.AcceptAllWriter(),
	})
	r := newTestRouter(t, Deps{SessionManager: mgr, Distiller: d})
	body := strings.NewReader(`{"user_id":"u1"}`)
	req := httptest.NewRequest(http.MethodPost, "/sessions/s1/distill", body)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var res session.Result
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	if res.Accepted == 0 {
		t.Fatalf("expected at least 1 accepted lesson, got %+v", res)
	}
}

func TestRollbackPipelineRun_Success(t *testing.T) {
	l := rollback.NewInMemoryLedger()
	_ = l.Record(context.Background(), rollback.LedgerEntry{
		PipelineRunID: "r1",
		NodeIDs:       []string{"n1", "n2"},
	})
	r := newTestRouter(t, Deps{Ledger: l})
	req := httptest.NewRequest(http.MethodPost, "/pipeline/runs/r1/rollback", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if v, ok := resp["rolled_back_nodes"].(float64); !ok || int(v) != 2 {
		t.Fatalf("expected 2 rolled_back_nodes, got %v", resp["rolled_back_nodes"])
	}
}

func TestRollbackPipelineRun_UnknownRun(t *testing.T) {
	l := rollback.NewInMemoryLedger()
	r := newTestRouter(t, Deps{Ledger: l})
	req := httptest.NewRequest(http.MethodPost, "/pipeline/runs/missing/rollback", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rec.Code)
	}
}

func TestSearchEnhanced_503WhenNotConfigured(t *testing.T) {
	r := newTestRouter(t, Deps{})
	req := httptest.NewRequest(http.MethodGet, "/search/enhanced?query=x", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestSearchEnhanced_RunsRetriever(t *testing.T) {
	s := &fakeSearcherAdapter{}
	r := newTestRouter(t, Deps{Retriever: retrieval.NewVectorRetriever(s, nil)})
	req := httptest.NewRequest(http.MethodGet, "/search/enhanced?query=hello", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["answer"] == nil {
		t.Fatal("missing answer")
	}
}

// fakeSearcherAdapter implements the internal retrieval memorySearcher
// surface using the proper types from internal/memory/types.
type fakeSearcherAdapter struct{}

func (f *fakeSearcherAdapter) SearchSemantic(ctx context.Context, q string, limit int) ([]types.MemoryResult, error) {
	return []types.MemoryResult{{MemoryID: "m1", Text: "hello", Score: 0.9}}, nil
}
func (f *fakeSearcherAdapter) SearchKeyword(ctx context.Context, q string, limit int) ([]types.MemoryResult, error) {
	return nil, nil
}
func (f *fakeSearcherAdapter) SearchEntities(ctx context.Context, entities []string, limit int) ([]types.MemoryResult, error) {
	return nil, nil
}
func (f *fakeSearcherAdapter) ExtractQueryEntities(ctx context.Context, q string) ([]string, error) {
	return nil, nil
}
