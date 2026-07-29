package telemetry

import "context"

type Shutdown func(context.Context) error

