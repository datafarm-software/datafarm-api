package telemetry

import (
	"context"
	"fmt"
	"runtime"
	"sync/atomic"
	"time"

	otelruntime "go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
)

type OtlpRecorder struct {
	mp               *sdkmetric.MeterProvider
	apiCounter       metric.Int64Counter
	activeUsersCount atomic.Int64
	requestLatency   metric.Float64Histogram
}

// NOTE: This uses insecure HTTP
func NewOtlpRecorder(res *resource.Resource, endpoint string) (*OtlpRecorder, Shutdown, error) {
	exporter, err := otlpmetrichttp.New(
		context.Background(), otlpmetrichttp.WithEndpoint(endpoint),
		otlpmetrichttp.WithInsecure())
	if err != nil {
		return nil, nil, fmt.Errorf("init exporter: %v", err)
	}
	reader := sdkmetric.NewPeriodicReader(exporter,
		sdkmetric.WithProducer(otelruntime.NewProducer()))
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(reader), sdkmetric.WithResource(res))
	o := &OtlpRecorder{mp: mp}
	name := "datafarm-api/telemetry/metering"
	if err = o.setup(name); err != nil {
		return nil, nil, fmt.Errorf("setup meter: %v", err)
	}
	return o, mp.Shutdown, nil
}

func (o *OtlpRecorder) setup(name string) (err error) {
	if err = o.setupApiCounter(name); err != nil {
		return fmt.Errorf("setupApiCounter: %v", err)
	}
	if err = o.setupActiveUsersGauge(name); err != nil {
		return fmt.Errorf("setupActiveUsersGauge: %v", err)
	}
	if err = o.setupMemoryGauge(name); err != nil {
		return fmt.Errorf("setupMemoryGauge: %v", err)
	}
	if err = o.setupRequestLatency(name); err != nil {
		return fmt.Errorf("setupRequestLatency: %v", err)
	}
	if err = o.setupUptimeGauge(name); err != nil {
		return fmt.Errorf("setupUptimeGauge: %v", err)
	}
	if err = otelruntime.Start(); err != nil {
		return fmt.Errorf("otelruntime Start: %v", err)
	}
	return nil
}

func (o *OtlpRecorder) setupApiCounter(name string) (err error) {
	meter := o.mp.Meter(name)
	o.apiCounter, err = meter.Int64Counter(
		name+".api.counter",
		metric.WithDescription("Number of API calls."),
		metric.WithUnit("{call}"),
	)
	if err != nil {
		err = fmt.Errorf("int64Counter: %v", err)
	}
	return err
}

func (o *OtlpRecorder) setupActiveUsersGauge(name string) (err error) {
	meter := o.mp.Meter(name)
	_, err = meter.Int64ObservableGauge(
		name+".active.users.gauge",
		metric.WithDescription("Active Users Gauge"),
		metric.WithUnit("{users}"),
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

var processStart = time.Now().Unix()

func (o *OtlpRecorder) setupUptimeGauge(name string) (err error) {
	meter := o.mp.Meter(name)
	_, err = meter.Int64ObservableGauge(
		name+".uptime",
		metric.WithDescription("Process Uptime"),
		metric.WithUnit("s"),
		metric.WithInt64Callback(func(_ context.Context, ob metric.Int64Observer) error {
			ob.Observe(processStart)
			return nil
		}),
	)
	if err != nil {
		err = fmt.Errorf("init gauge: %v", err)
	}
	return err
}

func (o *OtlpRecorder) setupRequestLatency(name string) (err error) {
	meter := o.mp.Meter(name)
	o.requestLatency, err = meter.Float64Histogram(
		name+".task.duration",
		metric.WithDescription("The duration of task execution."),
		metric.WithUnit("s"),
	)
	if err != nil {
		err = fmt.Errorf("init histogram: %v", err)
	}
	return err
}

func (o *OtlpRecorder) setupMemoryGauge(name string) (err error) {
	meter := o.mp.Meter(name)
	_, err = meter.Int64ObservableGauge(
		name+".memory.heap",
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

func (o *OtlpRecorder) RecordLatency(ctx context.Context, dur time.Duration) error {
	o.requestLatency.Record(ctx, float64(dur))
	return nil
}

func (o *OtlpRecorder) ActiveUsersCountAdd(i int) {
	o.activeUsersCount.Add(int64(i))
}

func (o *OtlpRecorder) CountApiRequest(ctx context.Context, i int, attr ...attribute.KeyValue) {
	o.apiCounter.Add(ctx, int64(i), metric.WithAttributes(attr...))
}
