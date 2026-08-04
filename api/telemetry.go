package api

import (
	"log"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humamux"
)

func (a *Api) RecordLatency(humaCtx huma.Context, next func(huma.Context)) {
	r, w := humamux.Unwrap(humaCtx)
	ctx := r.Context()
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		if err := a.MetricRecorder.RecordLatency(ctx, duration); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Internal error while recording latency of the request."))
			log.Printf("recordLatency: %v", err)
		}
	}()
	next(humaCtx)
}
