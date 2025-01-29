package api

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

var g_ctx context.Context

type metadataFetcher interface {
	Close() error
	GetMapValue(deviceId, mapKey string) (string, error)
	GetAttachedSensors(deviceId string) ([]string, error)
	GetQueryFields(attachedSensors []string) ([]string, error)
}

type dataFetcher interface {
	GetData(metadata Metadata, startTime, stopTime string) ([]byte, error)
}

type Metadata struct {
	DeviceId, Network, Company, QueryRange string
	QueryFields                            []string
}

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
		port:            port,
		metadataFetcher: mf,
		dataFetcher:     df,
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
	a.router.Handle("/device/{deviceId}", http.HandlerFunc(a.GetDataForDevice)).Methods("GET")
}

func (a *Api) GetDataForDevice(w http.ResponseWriter, r *http.Request) {
	//TODO validate deviceId against authorised user's company
	var metadata Metadata
	vars := mux.Vars(r)
	deviceId := vars["deviceId"]
	metadata.DeviceId = deviceId
	startTime := r.URL.Query().Get("start")
	stopTime := r.URL.Query().Get("stop")
	a.formatQueryRange(startTime, stopTime)
	var wg sync.WaitGroup
	var nwErr, cErr, qErr error
	wg.Add(1)
	go func() {
		defer wg.Done()
		network, err := a.metadataFetcher.GetMapValue(deviceId, "Network")
		if err != nil {
			nwErr = fmt.Errorf("error getting network: %v", err)
		}
		metadata.Network = network
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		company, err := a.metadataFetcher.GetMapValue(deviceId, "Company")
		if err != nil {
			cErr = fmt.Errorf("error getting company: %v", err)
		}
		metadata.Company = company
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		attachedSensors, err := a.metadataFetcher.GetAttachedSensors(deviceId)
		if err != nil {
			qErr = fmt.Errorf("error getting attached sensors: %v", err)
		}
		queryFields, err := a.metadataFetcher.GetQueryFields(attachedSensors)
		if err != nil {
			qErr = fmt.Errorf("error getting query fields: %v", err)
		}
		metadata.QueryFields = queryFields
	}()
	wg.Wait()
	if nwErr != nil {
		log.Println(nwErr)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if cErr != nil {
		log.Println(cErr)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if qErr != nil {
		log.Println(qErr)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	jsonData, err := a.dataFetcher.GetData(metadata, startTime, stopTime)
	if err != nil {
		log.Printf("error getting data: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	if _, err = w.Write(jsonData); err != nil {
		log.Printf("Error writing response: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func (a *Api) formatQueryRange(startTime, stopTime string) (string, error) {
	relativeRange := false
	if startTime == "" {
		return "", fmt.Errorf("no start time provided")
	}
	if _, err := time.Parse(time.RFC3339, startTime); err != nil {
		relativeRange = true
	}
	if _, err := time.Parse(time.RFC3339, startTime); err == nil && stopTime == "" {
		return "", fmt.Errorf("start time is rfc3339, but stop time is empty. cannot procede")
	}
	if !relativeRange {
		if _, err := time.Parse(time.RFC3339, stopTime); err != nil {
			return "", fmt.Errorf("Invalid RFC3339 stop timestamp: %v", err)
		}
		return fmt.Sprintf("start: %s, stop: %s", startTime, stopTime), nil
	}
	return fmt.Sprintf("start: %s", startTime), nil
}
