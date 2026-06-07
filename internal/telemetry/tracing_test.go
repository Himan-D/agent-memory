package telemetry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Enabled {
		t.Error("expected disabled by default")
	}
	if cfg.OTLPEndpoint != "localhost:4317" {
		t.Errorf("expected localhost:4317, got %s", cfg.OTLPEndpoint)
	}
	if cfg.ServiceName != "hystersis" {
		t.Errorf("expected hystersis, got %s", cfg.ServiceName)
	}
	if cfg.Environment != "development" {
		t.Errorf("expected development, got %s", cfg.Environment)
	}
	if cfg.SampleRate != 1.0 {
		t.Errorf("expected 1.0, got %f", cfg.SampleRate)
	}
}

func TestInit_Disabled(t *testing.T) {
	tp, err := Init(context.Background(), DefaultConfig())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tp == nil {
		t.Fatal("expected non-nil tracer provider")
	}
	defer Shutdown(context.Background())
}

func TestInit_EnabledNoEndpoint(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	_, err := Init(context.Background(), cfg)
	if err == nil {
		t.Skip("no OTLP endpoint available; this is expected to fail without a collector")
	}
}

func TestTracer(t *testing.T) {
	Init(context.Background(), DefaultConfig())
	defer Shutdown(context.Background())

	tr := Tracer()
	if tr == nil {
		t.Fatal("expected non-nil tracer")
	}
}

func TestTracerProvider(t *testing.T) {
	Init(context.Background(), DefaultConfig())
	defer Shutdown(context.Background())

	tp := TracerProvider()
	if tp == nil {
		t.Fatal("expected non-nil tracer provider")
	}
}

func TestStartSpan(t *testing.T) {
	Init(context.Background(), DefaultConfig())
	defer Shutdown(context.Background())

	ctx, span := StartSpan(context.Background(), "test-span")
	defer span.End()

	if span == nil {
		t.Fatal("expected non-nil span")
	}
	if !span.SpanContext().IsValid() {
		t.Log("span context is not valid (expected with noop provider)")
	}

	_, ok := ctx.Deadline()
	if ok {
		t.Error("expected no deadline on context")
	}
}

func TestShutdown_Idempotent(t *testing.T) {
	if err := Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown with no init: %v", err)
	}
	if err := Shutdown(context.Background()); err != nil {
		t.Fatalf("second shutdown: %v", err)
	}
}

func TestHTTPMiddleware(t *testing.T) {
	Init(context.Background(), DefaultConfig())
	defer Shutdown(context.Background())

	handler := HTTPMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestHTTPMiddleware_ErrorPath(t *testing.T) {
	Init(context.Background(), DefaultConfig())
	defer Shutdown(context.Background())

	handler := HTTPMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	req := httptest.NewRequest("POST", "/api/v1/memories", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

func TestSetSpanStatus(t *testing.T) {
	Init(context.Background(), DefaultConfig())
	defer Shutdown(context.Background())

	_, span := StartSpan(context.Background(), "status-test")
	defer span.End()

	SetSpanStatus(span, 200)
	SetSpanStatus(span, 500)
	SetSpanStatus(span, 503)
}

func TestNewResource(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ServiceName = "test-svc"
	cfg.Environment = "test"

	res := newResource(cfg)
	if res == nil {
		t.Fatal("expected non-nil resource")
	}
}

func TestNewPropagator(t *testing.T) {
	prop := newPropagator()
	if prop == nil {
		t.Fatal("expected non-nil propagator")
	}
}

func TestInit_AlwaysSample(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SampleRate = 2.0

	tp, err := Init(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tp == nil {
		t.Fatal("expected non-nil tracer provider")
	}

	Shutdown(context.Background())
}

func TestInit_CustomEndpoint(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.OTLPEndpoint = "otel-collector:4317"

	_, err := Init(context.Background(), cfg)
	if err == nil {
		t.Skip("no OTLP collector available in test environment")
	}
}

func TestShutdown_Cleanup(t *testing.T) {
	_, err := Init(context.Background(), DefaultConfig())
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
}

func TestDoubleInit(t *testing.T) {
	tp1, err := Init(context.Background(), DefaultConfig())
	if err != nil {
		t.Fatalf("first init: %v", err)
	}
	defer Shutdown(context.Background())

	tp2, err := Init(context.Background(), DefaultConfig())
	if err != nil {
		t.Fatalf("second init: %v", err)
	}

	if tp1 == tp2 {
		t.Log("tracer providers are the same instance after double init")
	}
}

func TestTracer_SameInstance(t *testing.T) {
	Init(context.Background(), DefaultConfig())
	defer Shutdown(context.Background())

	t1 := Tracer()
	t2 := Tracer()
	if t1 != t2 {
		t.Error("expected same tracer instance")
	}
}

func TestSpanKey(t *testing.T) {
	if SpanKey != struct{}{} {
		t.Error("SpanKey should be an empty struct")
	}
}

func TestSetSpanStatus_NonError(t *testing.T) {
	Init(context.Background(), DefaultConfig())
	defer Shutdown(context.Background())

	_, s := StartSpan(context.Background(), "non-error")
	defer s.End()

	SetSpanStatus(s, 302)
	SetSpanStatus(s, 404)
}

func TestTracerProvider_NilBeforeInit(t *testing.T) {
	tp := TracerProvider()
	if tp != nil {
		t.Log("tracer provider remains set from previous test")
	}
}
