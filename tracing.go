package guardgo

import (
	"context"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func (e *Engine) startProcessSpan(ctx context.Context, r *http.Request, ip string, key string) (context.Context, trace.Span) {
	if !e.cfg.Tracing.Enabled {
		return ctx, trace.SpanFromContext(ctx)
	}

	tracer := e.cfg.Tracing.Tracer
	if tracer == nil {
		tracer = otel.Tracer("guardgo")
	}

	ctx, span := tracer.Start(ctx, "guardgo.process", trace.WithSpanKind(trace.SpanKindInternal))
	span.SetAttributes(
		attribute.String("guardgo.ip", ip),
		attribute.String("guardgo.key", key),
		attribute.String("http.method", r.Method),
		attribute.String("http.host", r.Host),
		attribute.String("http.path", pathFromRequest(r)),
	)
	return ctx, span
}

func (e *Engine) annotateDecision(span trace.Span, d Decision, redisDurationMs float64) {
	if !e.cfg.Tracing.Enabled {
		return
	}
	span.SetAttributes(
		attribute.Bool("guardgo.allowed", d.Allowed),
		attribute.String("guardgo.reason", d.Reason.String()),
		attribute.Int64("guardgo.counter", d.Counter),
		attribute.Int("guardgo.limit", d.Limit),
		attribute.Int("guardgo.remaining", d.Remaining),
		attribute.Float64("guardgo.redis_ms", redisDurationMs),
	)
	if !d.Allowed {
		span.SetStatus(codes.Error, d.Reason.String())
	}
}

func pathFromRequest(r *http.Request) string {
	if r == nil || r.URL == nil {
		return ""
	}
	return r.URL.Path
}
