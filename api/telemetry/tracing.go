package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/contrib/exporters/autoexport"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func NewOtlpTracer(res *resource.Resource) (
	trace.Tracer, Shutdown, error) {
	exporter, err := autoexport.NewSpanExporter(context.Background())
	if err != nil {
		return nil, nil, fmt.Errorf("init exporter: %v", err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{},
	))
	tracer := tp.Tracer("datafarm-api/telemetry/tracing")
	return tracer, tp.Shutdown, nil
}
