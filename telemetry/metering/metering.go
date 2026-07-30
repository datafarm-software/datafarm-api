package metering

import (
	"context"
	"time"
)

type MetricProvider interface {
	RecordLatency(context.Context, time.Duration) error
	ApiCountAdd(context.Context, int)
	ActiveUsersCountAdd(int)
}
