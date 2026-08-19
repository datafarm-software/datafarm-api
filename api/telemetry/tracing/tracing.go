package tracing

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

type SpanKind trace.SpanKind

type Tracer interface {
	Close(context.Context) error
	Start(context.Context, string, SpanKind, map[string]string) (context.Context, Span)
}

type MockTracer struct{}

func (t *MockTracer) Close(context.Context) error { return nil }
func (t *MockTracer) Start(context.Context, string, SpanKind, map[string]string) (
	context.Context, Span) {
	return context.Background(), &MockSpan{}
}

type Span interface {
	End()
	SetAttributes(map[string]string)
}

type MockSpan struct{}

func (s *MockSpan) End()
func (s *MockSpan) SetAttributes(map[string]string)
