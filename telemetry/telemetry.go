package telemetry

import (
	"github.com/datafarm-software/datafarm-api/telemetry/logging"
	"github.com/datafarm-software/datafarm-api/telemetry/metering"
	"github.com/datafarm-software/datafarm-api/telemetry/tracing"
)

type Telemetry struct {
	logging.Logger
	metering.MetricProvider
	tracing.Tracer
}

func InitTelemetry() (t *Telemetry, err error) {
	t = &Telemetry{}
	t.Logger = &logging.OtlpLogger{}
	t.MetricProvider = &metering.OtlpMetering{}
	t.Tracer = &tracing.OtlpTracer{}
	return t, nil
}
