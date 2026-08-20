package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type OtlpTracer struct {
	trace.Tracer
	*sdktrace.TracerProvider
}

func NewOtlpTracer(res *resource.Resource, endpoint string) (
	Tracer, error) {
	exporter, err := otlptracehttp.New(
		context.Background(),
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("init exporter: %v", err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{},
	))
	tracer := &OtlpTracer{
		Tracer:         tp.Tracer("datafarm-api/telemetry/tracing"),
		TracerProvider: tp,
	}
	return tracer, nil
}

func (o *OtlpTracer) Close(ctx context.Context) error {
	return o.TracerProvider.Shutdown(ctx)
}

func (o *OtlpTracer) Start(ctx context.Context, name string, kind SpanKind,
	mapAttrs map[string]string) (retCtx context.Context, s Span) {
	attrs := make([]attribute.KeyValue, len(mapAttrs)+1)
	for k, v := range mapAttrs {
		attrs = append(attrs, attribute.String(k, v))
	}
	os := &OtlpSpan{}
	retCtx, os.Span = o.Tracer.Start(ctx, name,
		trace.WithSpanKind(trace.SpanKind(kind)),
		trace.WithAttributes(attrs...))
	return retCtx, os
}

func (o *OtlpTracer) SpanFromContext(ctx context.Context) (Span, error) {
	span := trace.SpanFromContext(ctx)
	var err error
	if !span.SpanContext().IsValid() {
		err = SpanNotFound
	}
	//NOTE: always able to return span because it is a no op if not found
	return &OtlpSpan{span}, err
}

type OtlpSpan struct {
	trace.Span
}

func (o *OtlpSpan) End() {}
func (o *OtlpSpan) SetAttributes(mapAttrs map[string]string) {
	attrs := make([]attribute.KeyValue, len(mapAttrs))
	for k, v := range mapAttrs {
		attrs = append(attrs, attribute.String(k, v))
	}
	o.Span.SetAttributes(attrs...)
}
