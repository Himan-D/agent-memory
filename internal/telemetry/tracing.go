package telemetry

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

const (
	instrumentationName = "agent-memory"
	tracerVersion       = "0.1.0"
)

var (
	tracerProvider *sdktrace.TracerProvider
	tracer         trace.Tracer
)

type Config struct {
	Enabled      bool
	OTLPEndpoint string
	ServiceName  string
	Environment  string
	SampleRate   float64
}

func DefaultConfig() Config {
	return Config{
		Enabled:      false,
		OTLPEndpoint: "localhost:4317",
		ServiceName:  "hystersis",
		Environment:  "development",
		SampleRate:   1.0,
	}
}

func Init(ctx context.Context, cfg Config) (*sdktrace.TracerProvider, error) {
	if !cfg.Enabled {
		tp := sdktrace.NewTracerProvider(
			sdktrace.WithSampler(sdktrace.NeverSample()),
			sdktrace.WithResource(newResource(cfg)),
		)
		otel.SetTracerProvider(tp)
		otel.SetTextMapPropagator(newPropagator())
		tracerProvider = tp
		tracer = tp.Tracer(instrumentationName, trace.WithInstrumentationVersion(tracerVersion))
		return tp, nil
	}

	exp, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("create otlp exporter: %w", err)
	}

	sampler := sdktrace.TraceIDRatioBased(cfg.SampleRate)
	if cfg.SampleRate >= 1.0 {
		sampler = sdktrace.AlwaysSample()
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.ParentBased(sampler)),
		sdktrace.WithBatcher(exp,
			sdktrace.WithBatchTimeout(5*time.Second),
			sdktrace.WithMaxExportBatchSize(512),
		),
		sdktrace.WithResource(newResource(cfg)),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(newPropagator())
	tracerProvider = tp
	tracer = tp.Tracer(instrumentationName, trace.WithInstrumentationVersion(tracerVersion))
	return tp, nil
}

func Shutdown(ctx context.Context) error {
	if tracerProvider == nil {
		return nil
	}
	return tracerProvider.Shutdown(ctx)
}

func Tracer() trace.Tracer {
	if tracer == nil {
		tp := otel.GetTracerProvider()
		tracer = tp.Tracer(instrumentationName, trace.WithInstrumentationVersion(tracerVersion))
	}
	return tracer
}

func TracerProvider() *sdktrace.TracerProvider {
	return tracerProvider
}

func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return Tracer().Start(ctx, name, opts...)
}

func newResource(cfg Config) *sdkresource.Resource {
	return sdkresource.NewWithAttributes(
		"https://opentelemetry.io/schemas/1.26.0",
		attribute.String("service.name", cfg.ServiceName),
		attribute.String("service.version", tracerVersion),
		attribute.String("deployment.environment", cfg.Environment),
	)
}

func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

var SpanKey = struct{}{}

func HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		spanName := r.Method + " " + r.URL.Path
		ctx, span := StartSpan(ctx, spanName,
			trace.WithSpanKind(trace.SpanKindServer),
		)
		defer span.End()

		span.SetAttributes(
			attribute.String("http.request.method", r.Method),
			attribute.String("url.path", r.URL.Path),
			attribute.String("url.full", r.URL.RequestURI()),
			attribute.String("user_agent.original", r.UserAgent()),
		)

		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}

func SetSpanStatus(span trace.Span, statusCode int) {
	if statusCode >= 500 {
		span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", statusCode))
	} else {
		span.SetStatus(codes.Unset, "")
	}
	span.SetAttributes(attribute.Int("http.response.status_code", statusCode))
}
