package api

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
)

var g_ctx context.Context

type ApiOpts struct {
	port string
}

func NewApiOpts(port string) (ApiOpts, error) {
	if port == "" {
		return ApiOpts{}, fmt.Errorf("port is nil")
	}
	return ApiOpts{
		port: port,
	}, nil
}

type Api struct {
	port            string
	wg              sync.WaitGroup
	server          *http.Server
	router          *mux.Router
	v1Routes        *mux.Router
	shutdownCtxFunc context.CancelFunc
}

func NewApi(opts ApiOpts) (*Api, error) {
	ctx, cancel := context.WithCancel(context.Background())
	g_ctx = ctx
	api := &Api{
		shutdownCtxFunc: cancel,
		router:          mux.NewRouter().PathPrefix("/api/v1").Subrouter(),
		port:            opts.port,
	}
	api.server = &http.Server{
		Addr:    opts.port,
		Handler: api.router,
	}
	api.registerRoutes()
	return api, nil
}

func (a *Api) startGoRoutine(routineToBeExecuted func(ctx context.Context)) {
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		routineToBeExecuted(g_ctx)
	}()
}

func (a *Api) Shutdown() {
	if err := a.server.Shutdown(g_ctx); err != nil {
		log.Fatalf("HTTP shutdown error: %v", err)
	}
	a.shutdownCtxFunc()
	a.wg.Wait()
	log.Println("All goroutines have finished.")
}

func (a *Api) StartHttpServer() {
	a.startGoRoutine(func(ctx context.Context) {
		if err := a.server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server error: %v", err)
		}
		log.Println("Stopped serving new connections.")
	})
}

func (a *Api) registerRoutes() {
	a.router.Handle("/device", http.HandlerFunc(a.GetDataForDevice))
}

func (a *Api) GetDataForDevice(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Gorilla!\n"))
}
