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
	"os"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humamux"
	"github.com/danielgtaylor/huma/v2/humacli"
	"github.com/geraud22/datafarm-api/authoriser"
	"github.com/geraud22/datafarm-api/datafetcher"
	"github.com/geraud22/datafarm-api/metadatafetcher"
	"github.com/geraud22/datafarm-api/redis"
	"github.com/golang-jwt/jwt"
	"github.com/gorilla/mux"
)

const EmptyPayloadLength int = 16

var ctx context.Context
var claimsKey string = "jwtClaims"
var QUERYFIELD_REGEX = regexp.MustCompile(`^[a-zA-Z0-9_\-\s:]*$`)
var DEVICE_ID_REGEX = regexp.MustCompile(`\w{1,30}`)
var RELATIVETIME_REGEX = regexp.MustCompile(`-\d{1,3}(?:[hdwy]|mo?)`)
var USERNAME_REGEX = regexp.MustCompile(`^[\w .@]{1,75}`)
var UPPERCASE_REGEX = regexp.MustCompile(`[A-Z]`)
var LOWERCASE_REGEX = regexp.MustCompile(`[a-z]`)
var NUMBER_REGEX = regexp.MustCompile(`[0-9]`)
var SPECIAL_CHARS_REGEX = regexp.MustCompile(`[@$!%*?&#]`)

type metadataFetcher interface {
	Close() error
	GetAttachedSensors(deviceId string) ([]string, error)
	GetQueryFields(attachedSensors []string) ([]string, error)
	GetCompany(deviceId string) (string, error)
	GetNetwork(deviceId string) (string, error)
}

type dataFetcher interface {
	GetData(metadata metadatafetcher.Metadata) (*datafetcher.ConsolidatedDeviceData, error)
	FormatQueryRange(startTime, stopTime string) (interface{}, error)
	Close() error
}

type tokenAuth interface {
	Close() error
	GenerateToken(userInfo authoriser.UserInfo) (string, error)
	GetPublicKey() *ecdsa.PublicKey
}

type basicAuth interface {
	Close() error
	CheckCredentials(username, passw string) (authoriser.UserInfo, error)
}

type ApiOpts struct {
	RedisOpts      redis.RedisOpts        `mapstructure:"Redis" validate:"required"`
	InfluxOpts     datafetcher.InfluxOpts `mapstructure:"Influx" validate:"required"`
	AdminRole      string                 `mapstructure:"adminRole" validate:"required,alphanum"`
	Port           string                 `mapstructure:"port" validate:"required"`
	PrivateKeyFile string                 `mapstructure:"privatekeyfile" validate:"required"`
	PublicKeyFile  string                 `mapstructure:"publickeyfile" validate:"required"`
}

type Api struct {
	port, adminRole string
	metadataFetcher metadataFetcher
	dataFetcher     dataFetcher
	tokenAuth       tokenAuth
	basicAuth       basicAuth
}

func Start(opts ApiOpts) error {
	redis, err := redis.NewRedis(opts.RedisOpts)
	if err != nil {
		return err
	}
	df, err := datafetcher.NewInfluxDatafetcher(opts.InfluxOpts)
	if err != nil {
		return fmt.Errorf("error init influx: %v", err)
	}
	tokenAuth, err := authoriser.NewJwtAuth(os.DirFS("."), opts.PrivateKeyFile, opts.PublicKeyFile)
	if err != nil {
		return fmt.Errorf("error initializing jwt authoriser: %v", err)
	}
	api := &Api{
		port:            opts.Port,
		metadataFetcher: redis, dataFetcher: df,
		tokenAuth: tokenAuth,
		basicAuth: redis,
	}
	api.adminRole = opts.AdminRole
	cli := humacli.New(func(hooks humacli.Hooks, options *ApiOpts) {
		router := mux.NewRouter().PathPrefix("/api/v1").Subrouter()
		router.Use(api.verifyJwt)
		config := huma.DefaultConfig("DataFarm SensorData API", "1.0.0")
		humaApi := humamux.New(router, config)
		api.RegisterHumaOperations(humaApi)
		server := &http.Server{
			Addr:    fmt.Sprintf(":%d", options.Port),
			Handler: router,
		}
		hooks.OnStart(func() {
			log.Println("Server started on ", server.Addr)
			if err := server.ListenAndServe(); err != nil {
				if !errors.Is(err, http.ErrServerClosed) {
					log.Fatalf("HTTP server error: %v", err)
				}
			}
		})
		hooks.OnStop(func() {
			api.Close()
			server.Shutdown(context.Background())
		})
	})
	cli.Run()
	return nil
}

func (a *Api) Close() {
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
	log.Println("Api shutdown.")
}

type HumaError struct {
	Schema string `json:"$schema" doc:"Link to Object Model."`
	Title  string `json:"title" doc:"Name associated with error code."`
	Status int    `json:"status" doc:"Http Status Code."`
	Detail string `json:"detail" doc:"Human Readable explanation of what went wrong."`
}

func (a *Api) RegisterHumaOperations(api huma.API) {
	registry := huma.NewMapRegistry("#/errors", huma.DefaultSchemaNamer)
	operation := huma.Operation{
		Method: "GET",
		Path:   "/device/{deviceId}",
		Parameters: []*huma.Param{
			{
				Name:            "deviceId",
				In:              "path",
				Description:     "Device Id to get data for.",
				Required:        true,
				AllowEmptyValue: false,
				Schema: &huma.Schema{
					Type:    "string",
					Pattern: "^[a-zA-Z0-9]+$",
				},
			},
		},
		Tags:        []string{"GET"},
		Summary:     "Get Sensor Data.",
		Description: "Clients can use this route to request data from a specific sensor using its device id.",
		RequestBody: &huma.RequestBody{},
		Responses: map[string]*huma.Response{
			"500": {
				Description: "Internal Server Error",
				Content: map[string]*huma.MediaType{
					"application/json": {
						Schema: huma.SchemaFromType(registry, reflect.TypeFor[HumaError]()),
						Example: HumaError{
							Schema: "http://localhost:3030/schemas/ErrorModel.json",
							Title:  "Internal Server Error",
							Status: 500,
							Detail: "Internal error while getting data for the device.",
						},
					}}},
			"400": {
				Description: "Bad Request",
				Content: map[string]*huma.MediaType{
					"application/json": {
						Schema: huma.SchemaFromType(registry, reflect.TypeFor[HumaError]()),
						Example: HumaError{
							Schema: "http://localhost:3030/schemas/ErrorModel.json",
							Title:  "Bad Request",
							Status: 400,
							Detail: "Invalid start time.",
						},
					},
				},
			},
		},
	}
	huma.Register(api, operation, a.GetDeviceData)
	operation = huma.Operation{
		Method:      "POST",
		Path:        "/login",
		Tags:        []string{"POST"},
		Summary:     "Login.",
		Description: "Clients can use this route to login and receive an active session token.",
		RequestBody: &huma.RequestBody{},
		Responses: map[string]*huma.Response{
			"500": {
				Description: "Internal Server Error",
				Content: map[string]*huma.MediaType{
					"application/json": {
						Schema: huma.SchemaFromType(registry, reflect.TypeFor[HumaError]()),
						Example: HumaError{
							Schema: "http://localhost:3030/schemas/ErrorModel.json",
							Title:  "Internal Server Error",
							Status: http.StatusInternalServerError,
							Detail: "Internal error logging in.",
						},
					}}},
			"400": {
				Description: "Bad Request",
				Content: map[string]*huma.MediaType{
					"application/json": {
						Schema: huma.SchemaFromType(registry, reflect.TypeFor[HumaError]()),
						Example: HumaError{
							Schema: "http://localhost:3030/schemas/ErrorModel.json",
							Title:  "Bad Request",
							Status: http.StatusBadRequest,
							Detail: "No auth header provided.",
						},
					},
				},
			},
			"401": {
				Description: "Unauthorized",
				Content: map[string]*huma.MediaType{
					"application/json": {
						Schema: huma.SchemaFromType(registry, reflect.TypeFor[HumaError]()),
						Example: HumaError{
							Schema: "http://localhost:3030/schemas/ErrorModel.json",
							Title:  "Unauthorized",
							Status: http.StatusUnauthorized,
							Detail: "Unknown username.",
						},
					},
				},
			},
		},
	}
	huma.Register(api, operation, a.Login)
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
		log.Printf("deviceid failed the regex")
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	startTime := r.FormValue("start")
	startTime = strings.TrimSpace(startTime)
	if !RELATIVETIME_REGEX.MatchString(startTime) {
		if _, err := time.Parse(time.RFC3339Nano, startTime); err != nil {
			log.Println("start time is invalid rfc")
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
	}
	stopTime := r.FormValue("stop")
	stopTime = strings.TrimSpace(stopTime)
	if stopTime != "" {
		if !RELATIVETIME_REGEX.MatchString(stopTime) {
			if _, err := time.Parse(time.RFC3339Nano, stopTime); err != nil {
				log.Println("stop time is invalid rfc")
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
		}
	}
	claims, ok := r.Context().Value(claimsKey).(jwt.MapClaims)
	if !ok {
		log.Println("no jwt claims")
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	username, ok := claims["username"].(string)
	if !ok {
		log.Println("no username claim.")
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
	}
	company, ok := claims["company"].(string)
	if !ok {
		log.Println("no company claims")
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	network, ok := claims["network"].(string)
	if !ok {
		log.Println("no network claims")
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	userRole, ok := claims["role"].(string)
	if !ok {
		log.Println("no role claim")
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	formattedQueryRange, err := a.dataFetcher.FormatQueryRange(startTime, stopTime)
	if err != nil {
		log.Printf("error formatting query range: %v", err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	//TODO: allow users to ask for multiple queryfields at once
	requestedQueryFields := r.URL.Query()["queryField"]
	for _, qf := range requestedQueryFields {
		if !QUERYFIELD_REGEX.MatchString(qf) {
			log.Println("queryField failed the regex")
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
	}
	queryFields := make([]string, 0)
	if len(requestedQueryFields) < 1 {
		log.Printf("%s: request did not specify a query field.", username)
		http.Error(w, "No query field specified.", http.StatusBadRequest)
	}
	if requestedQueryFields[0] == "all" {
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
		queryFields = append(queryFields, requestedQueryFields...)
	}
	metadata := metadatafetcher.Metadata{
		Company:     company,
		DeviceId:    deviceId,
		Network:     network,
		QueryRange:  formattedQueryRange,
		QueryFields: queryFields,
	}
	if strings.ToLower(userRole) == a.adminRole {
		company, err := a.metadataFetcher.GetCompany(deviceId)
		if err != nil {
			log.Printf("error getting company for admin request: %v", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		//NOTE: if deviceId belongs to other company than admin is assigned to by default
		if company != metadata.Company {
			network, err := a.metadataFetcher.GetNetwork(deviceId)
			if err != nil {
				log.Printf("error getting network for admin request: %v", err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
			metadata.Company = company
			metadata.Network = network
		}
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
		if err := claims.Valid(); err != nil {
			log.Printf("claims validation error: %v", err)
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

type LoginRequest struct {
	Auth string `header:"Authorization" required:"true"`
}

func (a *Api) Login(ctx context.Context,
	lr *LoginRequest) (*struct{ Token string }, error) {
	parts := strings.Split(lr.Auth, " ")
	if len(parts) != 2 || parts[0] != "Basic" {
		return nil, huma.Error400BadRequest(
			"Authorization header must follow the basic format: 'Basic base64(username:password)'")
	}
	authBytes, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, huma.Error500InternalServerError(
			"Internal error decoding given base64.")
	}
	authInfo := strings.Split(string(authBytes), ":")
	if len(authInfo) != 2 {
		return nil, huma.Error400BadRequest("Invalid Basic format provided.")
	}
	username := authInfo[0]
	if ok := USERNAME_REGEX.MatchString(username); !ok {
		return nil, huma.Error400BadRequest("Username failed the regex.")
	}
	password := authInfo[1]
	if ok := UPPERCASE_REGEX.MatchString(password); !ok {
		return nil, huma.Error400BadRequest(
			"Password failed the regex.")
	}
	if ok := LOWERCASE_REGEX.MatchString(password); !ok {
		return nil, huma.Error400BadRequest(
			"Password failed the regex.")
	}
	if ok := NUMBER_REGEX.MatchString(password); !ok {
		return nil, huma.Error400BadRequest(
			"Password failed the regex.")
	}
	if ok := SPECIAL_CHARS_REGEX.MatchString(password); !ok {
		return nil, huma.Error400BadRequest(
			"Password failed the regex.")
	}
	verifiedUserInfo, err := a.basicAuth.CheckCredentials(username, password)
	if err != nil {
		return nil, huma.Error401Unauthorized("Bad credentials provided.")
	}
	token, err := a.tokenAuth.GenerateToken(verifiedUserInfo)
	if err != nil {
		return nil, huma.Error500InternalServerError(
			"Internal error generating the jwt.")
	}
	return &struct{ Token string }{token}, nil
}
