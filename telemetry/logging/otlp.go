package logging

import (
	"context"
	"fmt"

	"github.com/datafarm-software/datafarm-api/telemetry"
	"go.opentelemetry.io/contrib/bridges/otelzap"
	"go.opentelemetry.io/contrib/exporters/autoexport"
	"go.opentelemetry.io/otel/sdk/log"
	"go.uber.org/zap"
)

func NewOtlpLogger(l *zap.Logger) (*zap.Logger, telemetry.Shutdown, error) {
	if l == nil {
		return nil, nil, fmt.Errorf("logger nil")
	}
	ctx := context.Background()
	otlpexp, err := autoexport.NewLogExporter(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("newLogExporter: %v", err)
	}
	lp := log.NewLoggerProvider(
		log.WithProcessor(log.NewBatchProcessor(otlpexp)),
	)
	l = zap.New(otelzap.NewCore("datafarm-rest-go",
		otelzap.WithLoggerProvider(lp)))
	return l, lp.Shutdown, nil
}
