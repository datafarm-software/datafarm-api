package metering

import (
	"context"
	"time"
)

type MetricRecorder interface {
	RecordLatency(context.Context, time.Duration) error
	CountApiRequest(context.Context, int)
	ActiveUsersCountAdd(int)
}
