package telemetry

type Opts struct {
	CollectorEndpoint string `mapstructure:"collectorendpoint" validate:"required"`
}
