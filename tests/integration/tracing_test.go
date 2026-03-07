package integration_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"guardgo"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestTracingCreatesSpanWithDecisionAttributes(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	defer func() { _ = tp.Shutdown(context.Background()) }()

	engine, err := guardgo.NewEngine(guardgo.MiddlewareConfig{
		Redis:       rdb,
		MaxRequests: 2,
		Window:      time.Second,
		Tracing: guardgo.TracingConfig{
			Enabled: true,
			Tracer:  tp.Tracer("guardgo-tests"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()

	req, _ := http.NewRequest(http.MethodGet, "http://example.com/trace", nil)
	req.RemoteAddr = "50.1.1.1:1000"
	_ = engine.Process(context.Background(), req)

	spans := recorder.Ended()
	if len(spans) == 0 {
		t.Fatalf("expected at least one span")
	}
	span := spans[len(spans)-1]
	if span.Name() != "guardgo.process" {
		t.Fatalf("unexpected span name: %s", span.Name())
	}

	attrs := attributesToMap(span.Attributes())
	if attrs["guardgo.reason"] != "allowed" {
		t.Fatalf("expected guardgo.reason=allowed, got %q", attrs["guardgo.reason"])
	}
	if attrs["http.path"] != "/trace" {
		t.Fatalf("expected http.path=/trace, got %q", attrs["http.path"])
	}
}

func attributesToMap(kv []attribute.KeyValue) map[string]string {
	out := make(map[string]string, len(kv))
	for _, item := range kv {
		out[string(item.Key)] = item.Value.Emit()
	}
	return out
}
