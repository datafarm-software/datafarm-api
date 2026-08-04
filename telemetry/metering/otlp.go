package metering

import (
	"context"
	"fmt"
	"runtime"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
)

type Otlp struct {
	mp               *sdkmetric.MeterProvider
	apiCounter       metric.Int64Counter
	activeUsersCount atomic.Int64
	requestLatency   metric.Float64Histogram
}

func NewOtlpMeter(res *resource.Resource) (MetricRecorder, error) {
	exporter, err := otlpmetrichttp.New(context.Background())
	if err != nil {
		return nil, fmt.Errorf("init exporter: %v", err)
	}
	reader := sdkmetric.NewPeriodicReader(exporter)
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(reader), sdkmetric.WithResource(res))
	o := &Otlp{mp: mp}
	ms := meterSetup{
		meterProvider: mp,
		name:          "datafarm-api/telemetry/metering",
	}
	if err = o.setup(ms); err != nil {
		return nil, fmt.Errorf("setup meter: %v", err)
	}
	return o, nil
}

type meterSetup struct {
	meterProvider *sdkmetric.MeterProvider
	name          string
}

func (o *Otlp) setup(ms meterSetup) (err error) {
	if err = o.setupApiCounter(ms); err != nil {
		return fmt.Errorf("setupApiCounter: %v", err)
	}
	if err = o.setupActiveUsersGauge(ms); err != nil {
		return fmt.Errorf("setupActiveUsersGauge: %v", err)
	}
	if err = o.setupMemoryGauge(ms); err != nil {
		return fmt.Errorf("setupMemoryGauge: %v", err)
	}
	if err = o.setupRequestLatency(ms); err != nil {
		return fmt.Errorf("setupRequestLatency: %v", err)
	}
	return nil
}

func (o *Otlp) setupApiCounter(ms meterSetup) (err error) {
	meter := ms.meterProvider.Meter(ms.name)
	o.apiCounter, err = meter.Int64Counter(
		ms.name+".api.counter",
		metric.WithDescription("Number of API calls."),
		metric.WithUnit("{call}"),
	)
	if err != nil {
		err = fmt.Errorf("int64Counter: %v", err)
	}
	return err
}

func (o *Otlp) setupActiveUsersGauge(ms meterSetup) (err error) {
	meter := ms.meterProvider.Meter(ms.name)
	_, err = meter.Int64ObservableGauge(
		ms.name+".active.users.gauge",
		metric.WithDescription("Active Users Gauge"),
		metric.WithUnit("By"),
		metric.WithInt64Callback(func(_ context.Context, ob metric.Int64Observer) error {
			ob.Observe(o.activeUsersCount.Load())
			return nil
		}),
	)
	if err != nil {
		err = fmt.Errorf("init gauge: %v", err)
	}
	return err
}

func (o *Otlp) setupRequestLatency(ms meterSetup) (err error) {
	meter := ms.meterProvider.Meter(ms.name)
	o.requestLatency, err = meter.Float64Histogram(
		ms.name+".task.duration",
		metric.WithDescription("The duration of task execution."),
		metric.WithUnit("s"),
	)
	if err != nil {
		err = fmt.Errorf("init histogram: %v", err)
	}
	return err
}

func (o *Otlp) setupMemoryGauge(ms meterSetup) (err error) {
	meter := ms.meterProvider.Meter(ms.name)
	_, err = meter.Int64ObservableGauge(
		ms.name+".memory.heap",
		metric.WithDescription("Memory usage of the allocated heap objects."),
		metric.WithUnit("By"),
		metric.WithInt64Callback(func(_ context.Context, o metric.Int64Observer) error {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			o.Observe(int64(m.HeapAlloc))
			return nil
		}),
	)
	if err != nil {
		return fmt.Errorf("init gauge: %v", err)
	}
	return err
}

func (o *Otlp) RecordLatency(ctx context.Context, dur time.Duration) error {
	return fmt.Errorf("not implemented")
}

func (o *Otlp) ActiveUsersCountAdd(i int) {
	o.activeUsersCount.Add(int64(i))
}

func (o *Otlp) CountApiRequest(ctx context.Context, i int) {
	o.apiCounter.Add(ctx, int64(i))
}
