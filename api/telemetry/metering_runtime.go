package telemetry

import (
	"context"
	"fmt"
	"runtime/metrics"

	"go.opentelemetry.io/otel/metric"
)

const goroutineCount = "/sched/goroutines:goroutines"

func (o *OtlpRecorder) setupRuntime(name string) (err error) {}

func (o *OtlpRecorder) setupGoroutineCount(name string) (err error) {
	sample := make([]metrics.Sample, 1)
	sample[0].Name = name
	metrics.Read(sample)
	meter := o.mp.Meter(name)
	_, err = meter.Float64ObservableGauge(
		name+".goroutines",
		metric.WithDescription("GoRoutine Count"),
		metric.WithUnit("{goroutines}"),
		metric.WithFloat64Callback(func(_ context.Context, ob metric.Float64Observer) error {
			ob.Observe(sample[0].Value.Float64())
			return nil
		}),
	)
	if err != nil {
		err = fmt.Errorf("init gauge: %v", err)
	}
	return err
}
