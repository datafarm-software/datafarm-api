package api

import (
	"log"
	"math"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humamux"
	"github.com/datafarm-software/datafarm-api/api/authstore"
	"github.com/gorilla/mux"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
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
	_, r, path := getPath(humaCtx)
	ctx := r.Context()
	a.Metric.CountApiRequest(ctx, 1,
		attribute.String("http.route", path),
		attribute.String("http.method", r.Method),
	)
	next(humaCtx)
}

func (a *Api) TraceRequest(humaCtx huma.Context, next func(huma.Context)) {
	_, r, path := getPath(humaCtx)
	ctx := otel.GetTextMapPropagator().Extract(
		r.Context(), propagation.HeaderCarrier(r.Header),
	)
	var span trace.Span
	ctx, span = a.Tracer.Start(
		ctx, path, trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(attribute.String("http.method", r.Method)),
	)
	next(huma.WithContext(humaCtx, ctx))
	span.End()
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}

type KnownUserRequest struct {
	authstore.UserInfo
}

func (a *Api) LogRequest(humaCtx huma.Context, next func(huma.Context)) {
	w, r, path := getPath(humaCtx)
	sw := &statusWriter{ResponseWriter: w}
	op := humaCtx.Operation()
	//NOTE: important to get span before NewContext call
	span := trace.SpanFromContext(humaCtx.Context())
	humaCtx = humamux.NewContext(op, r, sw)
	kur := &KnownUserRequest{}
	humaCtx = huma.WithValue(humaCtx, "log-known-user", kur)
	next(humaCtx)
	fields := []zap.Field{
		zap.String("http.method", r.Method),
		zap.String("http.route", path),
		zap.Int("http.status_code", sw.status),
	}
	if kur.Username != "" {
		fields = append(fields,
			zap.String("client.username", kur.Username),
			zap.String("client.company", kur.Company),
			zap.String("client.network", kur.Network),
		)
	}
	if span.SpanContext().IsValid() {
		fields = append(fields,
			zap.String("trace_id", span.SpanContext().TraceID().String()),
			zap.String("span_id", span.SpanContext().SpanID().String()),
		)
	}
	switch getFirstDigit(sw.status) {
	case 4:
		a.Logger.Warn("HTTP Client Error", fields...)
	case 5:
		a.Logger.Error("HTTP Internal Error", fields...)
	default:
		a.Logger.Info("HTTP Client Request", fields...)
	}
}

func getFirstDigit(n int) int {
	n = int(math.Abs(float64(n)))
	for n >= 10 {
		n /= 10
	}
	return n
}

func getPath(humaCtx huma.Context) (w http.ResponseWriter, r *http.Request, path string) {
	r, w = humamux.Unwrap(humaCtx)
	path = r.URL.Path
	route := mux.CurrentRoute(r)
	if route != nil {
		path, _ = route.GetPathTemplate()
	}
	return
}
