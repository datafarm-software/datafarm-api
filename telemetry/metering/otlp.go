package metering

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
)

type Otlp struct {
	mp *metric.MeterProvider
}

func NewOtlpMeter(res *resource.Resource) (MetricProvider, error) {
	exporter, err := otlpmetrichttp.New(context.Background())
	if err != nil {
		return nil, fmt.Errorf("init exporter: %v", err)
	}
	reader := metric.NewPeriodicReader(exporter)
	mp := metric.NewMeterProvider(
		metric.WithReader(reader), metric.WithResource(res))
	return &Otlp{mp: mp}, nil
}

func (o *Otlp) RecordLatency(dur time.Duration) error {
	return fmt.Errorf("not implemented")
}
