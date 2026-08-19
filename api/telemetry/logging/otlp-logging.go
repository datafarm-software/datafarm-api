package logging

import (
	"context"
	"fmt"

	"github.com/datafarm-software/datafarm-api/api/telemetry"
	"go.opentelemetry.io/contrib/bridges/otelzap"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.uber.org/zap"
)

func NewOtlpLogger(res *resource.Resource, endpoint string) (*zap.Logger, telemetry.Shutdown, error) {
	if res == nil {
		return nil, nil, fmt.Errorf("resource nil")
	}
	ctx := context.Background()
	otlpexp, err := otlploghttp.New(ctx, otlploghttp.WithEndpoint(endpoint),
		otlploghttp.WithInsecure())
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
