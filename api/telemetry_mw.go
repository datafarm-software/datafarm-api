package api

import (
	"log"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humamux"
	"github.com/gorilla/mux"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func (a *Api) RecordLatency(humaCtx huma.Context, next func(huma.Context)) {
	r, w := humamux.Unwrap(humaCtx)
	ctx := r.Context()
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		if err := a.Metric.RecordLatency(ctx, duration); err != nil {
			w.Write([]byte("Internal error while recording latency of the request. Please notify a system administrator."))
			log.Printf("recordLatency: %v", err)
		}
	}()
	next(humaCtx)
}

func (a *Api) CountApiRequest(humaCtx huma.Context, next func(huma.Context)) {
	r, path := getPath(humaCtx)
	ctx := r.Context()
	a.Metric.CountApiRequest(ctx, 1,
		attribute.String("http.route", path),
		attribute.String("http.method", r.Method),
	)
	next(humaCtx)
}

func (a *Api) TraceRequest(humaCtx huma.Context, next func(huma.Context)) {
	r, path := getPath(humaCtx)
	ctx := r.Context()
	_, span := a.Tracer.Start(
		ctx, path, trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attribute.String("http.method", r.Method)),
	)
	next(humaCtx)
	span.End()
}

func getPath(humaCtx huma.Context) (r *http.Request, path string) {
	r, _ = humamux.Unwrap(humaCtx)
	path = r.URL.Path
	route := mux.CurrentRoute(r)
	if route != nil {
		path, _ = route.GetPathTemplate()
	}
	return
}
