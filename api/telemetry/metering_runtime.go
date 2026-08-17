package telemetry

import (
	"context"
	"fmt"
	"log"
	"runtime/metrics"

	"go.opentelemetry.io/otel/metric"
)

var runtimeSamples = []metrics.Sample{
	{Name: "/sched/goroutines:goroutines"},
}

func (o *OtlpRecorder) setupRuntime(processName string) (err error) {
	metrics.Read(runtimeSamples)
	var sampleName string
	var value metrics.Value
	meter := o.mp.Meter(processName)
	for _, sample := range runtimeSamples {
		sampleName, value = sample.Name, sample.Value
		switch value.Kind() {
		case metrics.KindUint64:
			_, err = meter.Int64ObservableGauge(
				processName+sampleName,
				metric.WithInt64Callback(func(_ context.Context, ob metric.Int64Observer) error {
					ob.Observe(int64(sample.Value.Uint64()))
					return nil
				}),
			)
			if err != nil {
				err = fmt.Errorf("init gauge: %v", err)
			}
		case metrics.KindFloat64:
		case metrics.KindFloat64Histogram:
		case metrics.KindBad:
			return fmt.Errorf("%s returned metrics.KindBad", sampleName)
		default:
			log.Printf("%s, unexpected metric Kind: %v\n", sampleName, value.Kind())
		}
	}
	return nil
}

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
