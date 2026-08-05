package telemetry

import "context"

type Opts struct {
	MeterEndpoint string `mapstructure:"meterendpoint" validate:"required"`
}

type Shutdown func(context.Context) error
