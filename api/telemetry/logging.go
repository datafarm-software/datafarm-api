package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/contrib/bridges/otelzap"
	"go.opentelemetry.io/contrib/exporters/autoexport"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.uber.org/zap"
)

func NewOtlpLogger(res *resource.Resource) (*zap.Logger, Shutdown, error) {
	if res == nil {
		return nil, nil, fmt.Errorf("resource nil")
	}
	ctx := context.Background()
	otlpexp, err := autoexport.NewLogExporter(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("newLogExporter: %v", err)
	}
	lp := log.NewLoggerProvider(
		log.WithProcessor(log.NewBatchProcessor(otlpexp)),
		log.WithResource(res),
	)
	l := zap.New(otelzap.NewCore("datafarm-api",
		otelzap.WithLoggerProvider(lp)))
	return l, lp.Shutdown, nil
}
