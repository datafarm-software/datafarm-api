package api

import (
	"log"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humamux"
	"github.com/gorilla/mux"
	"go.opentelemetry.io/otel/attribute"
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
	r, _ := humamux.Unwrap(humaCtx)
	ctx := r.Context()
	route := mux.CurrentRoute(r)
	var path string
	if route != nil {
		path, _ = route.GetPathTemplate()
	}
	a.Metric.CountApiRequest(ctx, 1,
		attribute.String("http.route", path),
		attribute.String("http.method", r.Method),
	)
	next(humaCtx)
}
