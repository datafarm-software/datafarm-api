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

type metadataFetcher interface{}

type dataFetcher interface{}

type ApiOpts struct {
	port            string
	dataFetcher     dataFetcher
	metadataFetcher metadataFetcher
}

func NewApiOpts(port string, mf metadataFetcher, df dataFetcher) (ApiOpts, error) {
	if port == "" || mf == nil || df == nil {
		return ApiOpts{}, fmt.Errorf("not all options present")
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
	metadataFetcher metadataFetcher
	dataFetcher     dataFetcher
	shutdownCtxFunc context.CancelFunc
}

func NewApi(opts ApiOpts) (*Api, error) {
	ctx, cancel := context.WithCancel(context.Background())
	g_ctx = ctx
	api := &Api{
		shutdownCtxFunc: cancel,
		router:          mux.NewRouter().PathPrefix("/api/v1").Subrouter(),
		port:            opts.port,
		metadataFetcher: opts.metadataFetcher,
		dataFetcher:     opts.dataFetcher,
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
	a.router.Handle("/device/{deviceId}", http.HandlerFunc(a.GetDataForDevice))
}

func (a *Api) GetDataForDevice(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Gorilla!\n"))
}
