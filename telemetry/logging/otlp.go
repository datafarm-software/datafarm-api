package logging

import (
	"context"
	"fmt"

	"go.opentelemetry.io/contrib/bridges/otelzap"
	"go.opentelemetry.io/contrib/exporters/autoexport"
	"go.opentelemetry.io/otel/sdk/log"
	"go.uber.org/zap"
)

type OtlpLogger struct {
	*zap.Logger
}

func NewOtlpLogger() (*OtlpLogger, error) {
	o := &OtlpLogger{}
	ctx := context.Background()
	otlpexp, err := autoexport.NewLogExporter(ctx)
	if err != nil {
		return o, fmt.Errorf("newLogExporter: %v", err)
	}
	lp := log.NewLoggerProvider(
		log.WithProcessor(log.NewBatchProcessor(otlpexp)),
	)
	o.Logger = zap.New(otelzap.NewCore("datafarm-rest-go",
		otelzap.WithLoggerProvider(lp)))
	return o, nil
}
