package tracing

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel/trace"
)

var SpanNotFound = errors.New("no span in context")

type SpanKind trace.SpanKind

type Tracer interface {
	Close(context.Context) error
	Start(context.Context, string, SpanKind, map[string]string) (context.Context, Span)
	//NOTE: could return SpanNotFound
	SpanFromContext(context.Context) (Span, error)
}

type MockTracer struct{}

func (t *MockTracer) Close(context.Context) error { return nil }
func (t *MockTracer) Start(context.Context, string, SpanKind, map[string]string) (
	context.Context, Span) {
	return context.Background(), &MockSpan{}
}

func (t *MockTracer) SpanFromContext(ctx context.Context) (Span, error) {
	return &MockSpan{}, nil
}

type Span interface {
	End()
	SetAttributes(map[string]string)
}

type MockSpan struct{}

func (s *MockSpan) End()                            {}
func (s *MockSpan) SetAttributes(map[string]string) {}
