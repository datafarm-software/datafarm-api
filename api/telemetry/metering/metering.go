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

type MockMeter struct{}

func (m *MockMeter) Close(context.Context) error                                        { return nil }
func (m *MockMeter) RecordLatency(ctx context.Context, dur time.Duration) error         { return nil }
func (m *MockMeter) ActiveUsersCountAdd(i int)                                          {}
func (m *MockMeter) CountApiRequest(ctx context.Context, i int, attr map[string]string) {}
