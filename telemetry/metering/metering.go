package metering

import (
	"context"
	"time"
)

type MetricRecorder interface {
	RecordLatency(context.Context, time.Duration) error
	ApiCountAdd(context.Context, int)
	ActiveUsersCountAdd(int)
}
