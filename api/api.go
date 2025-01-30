package api

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/gorilla/mux"
)

const EmptyPayloadLength int = 16

var g_ctx context.Context

type metadataFetcher interface {
	Close() error
	GetMapValue(deviceId, mapKey string) (string, error)
	GetAttachedSensors(deviceId string) ([]string, error)
	GetQueryFields(attachedSensors []string) ([]string, error)
}

type dataFetcher interface {
	GetData(metadata Metadata) ([]byte, error)
	Close() error
}

type authoriser interface {
	GenerateJwt() (string, error)
	GetPublicKey() *ecdsa.PublicKey
}

type Metadata struct {
	DeviceId, Network, Company, QueryRange string
	QueryFields                            []string
}

type ApiOpts struct {
	port            string
	dataFetcher     dataFetcher
	metadataFetcher metadataFetcher
	authoriser      authoriser
}

func NewApiOpts(port string, mf metadataFetcher, df dataFetcher, au authoriser) (ApiOpts, error) {
	if port == "" || mf == nil || df == nil || au == nil {
		return ApiOpts{}, fmt.Errorf("not all options present")
	}
	return ApiOpts{
		port:            port,
		metadataFetcher: mf,
		dataFetcher:     df,
		authoriser:      au,
	}, nil
}

type Api struct {
	port            string
	wg              sync.WaitGroup
	server          *http.Server
	router          *mux.Router
	metadataFetcher metadataFetcher
	dataFetcher     dataFetcher
	authoriser      authoriser
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
		authoriser:      opts.authoriser,
	}
	api.server = &http.Server{
		Addr:    opts.port,
		Handler: api.router,
	}
	api.registerRoutes()
	api.router.Use(api.verifyJwt)
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
	if err := a.dataFetcher.Close(); err != nil {
		log.Fatalf("datafetcher close error: %v", err)
	}
	if err := a.metadataFetcher.Close(); err != nil {
		log.Fatalf("metadatafetcher close error: %v", err)
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
	a.router.Handle("/login", http.HandlerFunc(a.Login)).Methods("POST")
}

func (a *Api) GetDataForDevice(w http.ResponseWriter, r *http.Request) {
	//TODO validate deviceId against authorised user's company
	var metadata Metadata
	vars := mux.Vars(r)
	deviceId := vars["deviceId"]
	deviceId = strings.TrimSpace(deviceId)
	metadata.DeviceId = deviceId
	startTime := r.URL.Query().Get("start")
	stopTime := r.URL.Query().Get("stop")
	var wg sync.WaitGroup
	var tErr, nwErr, cErr, aErr, qErr error
	wg.Add(1)
	go func() {
		defer wg.Done()
		queryRange, err := a.formatQueryRange(startTime, stopTime)
		if err != nil {
			tErr = fmt.Errorf("error formatting query range: %v", err)
			return
		}
		metadata.QueryRange = queryRange
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		network, err := a.metadataFetcher.GetMapValue(deviceId, "Network")
		if err != nil {
			nwErr = fmt.Errorf("error getting network: %v", err)
			return
		}
		metadata.Network = network
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		company, err := a.metadataFetcher.GetMapValue(deviceId, "Company")
		if err != nil {
			cErr = fmt.Errorf("error getting company: %v", err)
			return
		}
		metadata.Company = company
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		attachedSensors, err := a.metadataFetcher.GetAttachedSensors(deviceId)
		if err != nil {
			aErr = fmt.Errorf("error getting attached sensors: %v", err)
			return
		}
		queryFields, err := a.metadataFetcher.GetQueryFields(attachedSensors)
		if err != nil {
			qErr = fmt.Errorf("error getting query fields: %v", err)
			return
		}
		metadata.QueryFields = queryFields
	}()
	wg.Wait()
	if tErr != nil {
		log.Println(tErr)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	if nwErr != nil {
		log.Println(nwErr)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	if cErr != nil {
		log.Println(cErr)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	if qErr != nil {
		log.Println(qErr)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if aErr != nil {
		log.Println(aErr)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	jsonData, err := a.dataFetcher.GetData(metadata)
	if err != nil {
		log.Printf("error getting data: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	var bytesToReturn []byte
	if len(jsonData) > EmptyPayloadLength {
		bytesToReturn = jsonData
	} else {
		bytesToReturn = []byte(`{"null"}`)
	}
	if _, err := w.Write(bytesToReturn); err != nil {
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

func (a *Api) verifyJwt(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/login" {
			next.ServeHTTP(w, r)
			return
		}
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			log.Println("no auth header provided")
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			log.Println("Invalid Authorization header format")
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		tokenString := parts[1]
		token, err := jwt.ParseWithClaims(tokenString, jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodECDSA); !ok {
				log.Println("wrong signing method used")
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return nil, nil
			}
			return a.authoriser.GetPublicKey(), nil
		})
		if err != nil {
			log.Printf("token parsing error: %v", err)
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		if !token.Valid {
			log.Println("Invalid token provided")
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *Api) Login(w http.ResponseWriter, r *http.Request) {
	token, err := a.authoriser.GenerateJwt()
	if err != nil {
		log.Printf("error generating jwt: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/plain")
	if _, err := w.Write([]byte(token)); err != nil {
		log.Printf("error writing to response writer: %v", err)
		return
	}
}
