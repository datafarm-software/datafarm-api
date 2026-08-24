package logging

import (
	"context"
	"fmt"

	"go.opentelemetry.io/contrib/bridges/otelzap"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.uber.org/zap"
)

type OtlpLogger struct {
	*zap.Logger
	*log.LoggerProvider
}

func NewOtlpLogger(res *resource.Resource, endpoint string) (
	Logger, error) {
	if res == nil {
		return nil, fmt.Errorf("resource nil")
	}
	ctx := context.Background()
	otlpexp, err := otlploghttp.New(ctx, otlploghttp.WithEndpoint(endpoint),
		otlploghttp.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("newLogExporter: %v", err)
	}
	lp := log.NewLoggerProvider(
		log.WithProcessor(log.NewBatchProcessor(otlpexp)),
		log.WithResource(res),
	)
	l := &OtlpLogger{
		Logger: zap.New(otelzap.NewCore("datafarm-api",
			otelzap.WithLoggerProvider(lp))),
		LoggerProvider: lp,
	}
	return l, nil
}

func (o *OtlpLogger) Close(ctx context.Context) error {
	return o.LoggerProvider.Shutdown(ctx)
}

func makeFields(metadata Metadata) []zap.Field {
	attrs := make([]zap.Field, 0, len(metadata.KeyValue))
	var field zap.Field
	var le int
	for k, strSlice := range metadata.KeyValue {
		le = len(strSlice)
		switch {
		case le == 1:
			field = zap.String(k, strSlice[0])
		case le > 1:
			field = zap.Strings(k, strSlice)
		default:
			continue
		}
		attrs = append(attrs, field)
	}
	return attrs
}

func (o *OtlpLogger) Info(msg string, metadata Metadata) {
	o.Logger.Info(msg, makeFields(metadata)...)
}

func (o *OtlpLogger) Warn(msg string, metadata Metadata) {
	o.Logger.Warn(msg, makeFields(metadata)...)
}

func (o *OtlpLogger) Error(msg string, metadata Metadata) {
	o.Logger.Error(msg, makeFields(metadata)...)
}
