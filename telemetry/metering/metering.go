package metering

import "time"

type MetricProvider interface {
	RecordLatency(time.Duration) error
}
