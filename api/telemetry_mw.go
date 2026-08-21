package api

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humamux"
	"github.com/datafarm-software/datafarm-api/api/telemetry/logging"
	"github.com/datafarm-software/datafarm-api/api/telemetry/tracing"
	"github.com/gorilla/mux"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func (a *Api) RecordLatency(humaCtx huma.Context, next func(huma.Context)) {
	r, w := humamux.Unwrap(humaCtx)
	ctx := r.Context()
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		if err := a.Meter.RecordLatency(ctx, duration); err != nil {
			w.Write([]byte("Internal error while recording latency of the request. Please notify a system administrator."))
			log.Printf("recordLatency: %v", err)
		}
	}()
	next(humaCtx)
}

func (a *Api) CountApiRequest(humaCtx huma.Context, next func(huma.Context)) {
	path := getPath(humaCtx)
	a.Meter.CountApiRequest(humaCtx.Context(), 1,
		map[string]string{
			"http.route":  path,
			"http.method": humaCtx.Method(),
		},
	)
	next(humaCtx)
}

func (a *Api) TraceRequest(humaCtx huma.Context, next func(huma.Context)) {
	path := getPath(humaCtx)
	headerMap := make(map[string]string)
	humaCtx.EachHeader(func(name, value string) {
		headerMap[name] = value
	})
	ctx := otel.GetTextMapPropagator().Extract(
		humaCtx.Context(), propagation.MapCarrier(headerMap),
	)
	var span tracing.Span
	ctx, span = a.Tracer.Start(
		ctx, path, tracing.SpanKind(trace.SpanKindServer),
		map[string]string{"http.method": humaCtx.Method()},
	)
	next(huma.WithContext(humaCtx, ctx))
	span.End()
}

type requestLog struct {
	logging.Metadata
}

func (a *Api) LogRequest(humaCtx huma.Context, next func(huma.Context)) {
	rl := &requestLog{logging.Metadata{
		KeyValue: map[string][]string{
			"http.method": {humaCtx.Method()},
			"http.route":  {getPath(humaCtx)},
		},
	}}
	span, _ := a.Tracer.SpanFromContext(humaCtx.Context())
	if span.IsValid() {
		rl.KeyValue["trace_id"] = []string{span.TraceId()}
		rl.KeyValue["span_id"] = []string{span.SpanId()}
	}
	humaCtx = huma.WithValue(humaCtx, "request-log", rl)
	next(humaCtx)
	rl.KeyValue["http.status_code"] = []string{fmt.Sprintf("%d", humaCtx.Status())}
	switch getFirstDigit(humaCtx.Status()) {
	case 4:
		a.Logger.Warn("HTTP Client Error", rl.Metadata)
	case 5:
		a.Logger.Error("HTTP Internal Error", rl.Metadata)
	default:
		a.Logger.Info("HTTP Client Request", rl.Metadata)
	}
}

func getFirstDigit(n int) int {
	n = int(math.Abs(float64(n)))
	for n >= 10 {
		n /= 10
	}
	return n
}

func getPath(humaCtx huma.Context) (path string) {
	r, _ := humamux.Unwrap(humaCtx)
	path = r.URL.Path
	route := mux.CurrentRoute(r)
	if route != nil {
		path, _ = route.GetPathTemplate()
	}
	return
}
