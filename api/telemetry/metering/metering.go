package metering

import (
	"context"
	"time"
)

type Meter interface {
	Close(context.Context) error
	RecordLatency(ctx context.Context, dur time.Duration) error
	ActiveUsersCountAdd(i int)
	CountApiRequest(ctx context.Context, i int, attr map[string]string)
}
