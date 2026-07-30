package metering

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
)

func NewOtlpMeter(res *resource.Resource) (*metric.MeterProvider, error) {
	exporter, err := otlpmetrichttp.New(context.Background())
	if err != nil {
		return nil, fmt.Errorf("init exporter: %v", err)
	}
	reader := metric.NewPeriodicReader(exporter)
	mp := metric.NewMeterProvider(
		metric.WithReader(reader), metric.WithResource(res))
	return mp, nil
}
