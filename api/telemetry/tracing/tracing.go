package tracing

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

type SpanKind trace.SpanKind

type Tracer interface {
	Start(context.Context, string, SpanKind, map[string]string) (context.Context, Span)
}

type Span interface {
	End()
	SetAttributes(map[string]string)
}
