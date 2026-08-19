package telemetry

import "context"

type Opts struct {
	CollectorEndpoint string `mapstructure:"collectorendpoint" validate:"required"`
}

type Shutdown func(context.Context) error
