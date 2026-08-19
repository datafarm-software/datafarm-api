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

type OtlpLogger struct {
	*zap.Logger
}

func NewOtlpLogger(res *resource.Resource, endpoint string) (
	Logger, telemetry.Shutdown, error) {
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
	l := &OtlpLogger{
		zap.New(otelzap.NewCore("datafarm-api",
			otelzap.WithLoggerProvider(lp))),
	}
	return l, lp.Shutdown, nil
}

func makeFields(metadata map[string]string) []zap.Field {
	attrs := make([]zap.Field, len(metadata))
	for k, v := range metadata {
		attrs = append(attrs, zap.String(k, v))
	}
	return attrs
}

func (o *OtlpLogger) Info(msg string, metadata map[string]string) {
	o.Logger.Info(msg, makeFields(metadata)...)
}

func (o *OtlpLogger) Warn(msg string, metadata map[string]string) {
	o.Logger.Warn(msg, makeFields(metadata)...)
}

func (o *OtlpLogger) Error(msg string, metadata map[string]string) {
	o.Logger.Error(msg, makeFields(metadata)...)
}
