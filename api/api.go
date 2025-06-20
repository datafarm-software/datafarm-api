package api

import (
	"context"
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/gorilla/mux"
)

const EmptyPayloadLength int = 16

var g_ctx context.Context
var claimsKey string = "jwtClaims"
var QUERYFIELD_REGEX = regexp.MustCompile(`^[a-zA-Z0-9_\-\s:]*$`)
var DEVICE_ID_REGEX = regexp.MustCompile(`\w{30}`)
var RELATIVETIME_REGEX = regexp.MustCompile(`-\d{1,3}(?:[hdwy]|mo?)`)

type metadataFetcher interface {
	Close() error
	GetAttachedSensors(deviceId string) ([]string, error)
	GetQueryFields(attachedSensors []string) ([]string, error)
}

type dataFetcher interface {
	GetData(metadata Metadata) (*ConsolidatedDeviceData, error)
	FormatQueryRange(startTime, stopTime string) (interface{}, error)
	Close() error
}

type tokenAuth interface {
	Close() error
	GenerateToken(userInfo UserInfo) (string, error)
	GetPublicKey() *ecdsa.PublicKey
}

type basicAuth interface {
	Close() error
	CheckCredentials(username, passw string) (UserInfo, error)
}

type ConsolidatedDeviceData struct {
	DeviceData []DeviceData `json:"payload"`
}

type DeviceData struct {
	DeviceID   string    `json:"rtuid"`
	Timestamp  time.Time `json:"timestamp"`
	SensorData map[string]interface{}
}

type Metadata struct {
	DeviceId, Company, Network string
	QueryRange                 interface{}
	QueryFields                []string
}

type UserInfo struct {
	Username, Company, Network string
}

type ApiOpts struct {
	port            string
	dataFetcher     dataFetcher
	metadataFetcher metadataFetcher
	tokenAuth       tokenAuth
	basicAuth       basicAuth
}

func NewApiOpts(port string, mf metadataFetcher, df dataFetcher, t tokenAuth, b basicAuth) (ApiOpts, error) {
	if port == "" || mf == nil || df == nil || t == nil || b == nil {
		return ApiOpts{}, fmt.Errorf("not all options present")
	}
	return ApiOpts{
		port:            port,
		metadataFetcher: mf,
		dataFetcher:     df,
		tokenAuth:       t,
		basicAuth:       b,
	}, nil
}

type Api struct {
	port            string
	wg              sync.WaitGroup
	server          *http.Server
	router          *mux.Router
	metadataFetcher metadataFetcher
	dataFetcher     dataFetcher
	tokenAuth       tokenAuth
	basicAuth       basicAuth
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
		tokenAuth:       opts.tokenAuth,
		basicAuth:       opts.basicAuth,
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
	if err := a.metadataFetcher.Close(); err != nil {
		log.Fatalf("error closing metadatafetcher: %v", err)
	}
	if err := a.dataFetcher.Close(); err != nil {
		log.Fatalf("error closing datafetcher: %v", err)
	}
	if err := a.tokenAuth.Close(); err != nil {
		log.Fatalf("error closing token auth: %v", err)
	}
	if err := a.basicAuth.Close(); err != nil {
		log.Fatalf("error closing basic auth: %v", err)
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
	a.router.Handle("/device/{deviceId}", http.HandlerFunc(a.GetDeviceData)).Methods("GET")
	a.router.Handle("/login", http.HandlerFunc(a.Login)).Methods("GET", "POST")
}

func (a *Api) GetDeviceData(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		log.Println("error parsing form while loading dashboard: %v", err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	routeVars := mux.Vars(r)
	deviceId := routeVars["deviceId"]
	deviceId = strings.TrimSpace(deviceId)
	if !DEVICE_ID_REGEX.MatchString(deviceId) {
		log.Println("deviceid failed the regex")
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	startTime := routeVars["start"]
	startTime = strings.TrimSpace(startTime)
	if !RELATIVETIME_REGEX.MatchString(startTime) {
		if _, err := time.Parse(time.RFC3339Nano, startTime); err != nil {
			log.Println("start time is invalid rfc")
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
	}
	stopTime := routeVars["stop"]
	stopTime = strings.TrimSpace(stopTime)
	if !RELATIVETIME_REGEX.MatchString(stopTime) {
		if _, err := time.Parse(time.RFC3339Nano, stopTime); err != nil {
			log.Println("start time is invalid rfc")
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
	}
	//TODO: allow users to ask for multiple queryfields at once
	requestedQueryField := routeVars["queryField"]
	requestedQueryField = strings.TrimSpace(requestedQueryField)
	if !QUERYFIELD_REGEX.MatchString(requestedQueryField) {
		log.Println("queryField failed the regex")
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	claims, ok := r.Context().Value(claimsKey).(jwt.MapClaims)
	if !ok {
		log.Println("no jwt claims")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	company, ok := claims["company"].(string)
	if !ok {
		log.Println("no company claims")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	network, ok := claims["network"].(string)
	if !ok {
		log.Println("no network claims")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	formattedQueryRange, err := a.dataFetcher.FormatQueryRange(startTime, stopTime)
	if err != nil {
		log.Printf("error formatting query range: %v", err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	queryFields := make([]string, 0)
	if requestedQueryField == "" || requestedQueryField == "all" {
		attachedSensors, err := a.metadataFetcher.GetAttachedSensors(deviceId)
		if err != nil {
			log.Printf("error getting attached sensors: %v", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		qf, err := a.metadataFetcher.GetQueryFields(attachedSensors)
		if err != nil {
			log.Printf("error getting query fields: %v", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		queryFields = qf
	} else {
		queryFields = append(queryFields, requestedQueryField)
	}
	metadata := Metadata{
		Company:     company,
		DeviceId:    deviceId,
		Network:     network,
		QueryRange:  formattedQueryRange,
		QueryFields: queryFields,
	}
	deviceData, err := a.dataFetcher.GetData(metadata)
	if err != nil {
		log.Printf("error getting data: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	jsonData, err := json.Marshal(deviceData)
	if err != nil {
		log.Printf("error marshalling deviceData to json: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	var bytesToReturn []byte
	if len(jsonData) > EmptyPayloadLength {
		bytesToReturn = jsonData
	} else {
		bytesToReturn = []byte(`{"payload": []}`)
	}
	if _, err := w.Write(bytesToReturn); err != nil {
		log.Printf("Error writing response: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
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
		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodECDSA); !ok {
				log.Println("wrong signing method used")
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return nil, nil
			}
			return a.tokenAuth.GetPublicKey(), nil
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
		ctx := context.WithValue(r.Context(), claimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (a *Api) Login(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		log.Println("no auth header provided")
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Basic" {
		log.Println("Invalid Authorization header format")
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	authBytes, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		log.Printf("error decoding given base64: %v", err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	authInfo := strings.Split(string(authBytes), ":")
	if len(authInfo) != 2 {
		log.Println("Invalid Basic format provided")
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	username := authInfo[0]
	password := authInfo[1]
	verifiedUserInfo, err := a.basicAuth.CheckCredentials(username, password)
	if err != nil {
		log.Printf("error checking credentials: %v", err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	token, err := a.tokenAuth.GenerateToken(verifiedUserInfo)
	if err != nil {
		log.Printf("error generating jwt: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write([]byte(token)); err != nil {
		log.Printf("error writing to response writer: %v", err)
		return
	}
}
