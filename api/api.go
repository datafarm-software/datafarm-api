package api

import (
	"context"
	"encoding/base64"
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
	"github.com/geraud22/datafarm-api/authstore"
	"github.com/geraud22/datafarm-api/datafetcher"
	df "github.com/geraud22/datafarm-api/datafetcher"
	deviceinfo "github.com/geraud22/datafarm-api/device-info"
	"github.com/geraud22/datafarm-api/tokenprovider"

	"github.com/geraud22/datafarm-api/redis"
	"github.com/gorilla/mux"
)

const EmptyPayloadLength int = 16

var ctx context.Context
var QUERYFIELD_REGEX = regexp.MustCompile(`^[a-zA-Z0-9_\-\s:]*$`)

// var DEVICE_ID_REGEX = regexp.MustCompile(`\w{1,30}`)
var RELATIVETIME_REGEX = regexp.MustCompile(`-\d{1,3}(?:[hdwy]|mo?)`)
var USERNAME_REGEX = regexp.MustCompile(`^[\w .@]{1,75}`)
var UPPERCASE_REGEX = regexp.MustCompile(`[A-Z]`)
var LOWERCASE_REGEX = regexp.MustCompile(`[a-z]`)
var NUMBER_REGEX = regexp.MustCompile(`[0-9]`)
var SPECIAL_CHARS_REGEX = regexp.MustCompile(`[@$!%*?&#]`)

type ApiOpts struct {
	RedisOpts      redis.RedisOpts        `mapstructure:"Redis" validate:"required"`
	InfluxOpts     datafetcher.InfluxOpts `mapstructure:"Influx" validate:"required"`
	AdminRole      string                 `mapstructure:"adminRole" validate:"required,alphanum"`
	Port           string                 `mapstructure:"port" validate:"required"`
	PrivateKeyFile string                 `mapstructure:"privatekeyfile" validate:"required"`
	PublicKeyFile  string                 `mapstructure:"publickeyfile" validate:"required"`
}

type Api struct {
	Port, AdminRole string
	DeviceInfo      deviceinfo.DeviceInfoFetcher
	DataFetcher     df.DataFetcher
	TokenProvider   tokenprovider.TokenProvider
	AuthStore       authstore.AuthStore
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
	tokenAuth, err := tokenprovider.NewJwtAuth(os.DirFS("."), opts.PrivateKeyFile, opts.PublicKeyFile)
	if err != nil {
		return fmt.Errorf("error initializing jwt authstore: %v", err)
	}
	api := &Api{
		Port:       opts.Port,
		DeviceInfo: redis, DataFetcher: df,
		TokenProvider: tokenAuth,
		AuthStore:     redis,
	}
	api.AdminRole = opts.AdminRole
	cli := humacli.New(func(hooks humacli.Hooks, options *ApiOpts) {
		router := mux.NewRouter().PathPrefix("/api/v1").Subrouter()
		config := huma.DefaultConfig("SensorData API", "1.0.0")
		config.Servers = append(config.Servers, &huma.Server{URL: "/api/v1"})
		humaApi := humamux.New(router, config)
		api.RegisterHumaOperations(humaApi)
		server := &http.Server{
			Addr:    opts.Port,
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
	if err := a.DeviceInfo.Close(); err != nil {
		log.Fatalf("error closing metadatafetcher: %v", err)
	}
	if err := a.DataFetcher.Close(); err != nil {
		log.Fatalf("error closing datafetcher: %v", err)
	}
	if err := a.TokenProvider.Close(); err != nil {
		log.Fatalf("error closing token auth: %v", err)
	}
	if err := a.AuthStore.Close(); err != nil {
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
		Method:      "GET",
		Path:        "/device/{deviceId}",
		Middlewares: huma.Middlewares{a.verifyToken},
		Parameters: []*huma.Param{
			{
				Name:        "Authorization",
				In:          "header",
				Description: "Token used for authentication. Note 'Bearer' prefix.",
				Required:    true,
				Schema: &huma.Schema{
					Type:    "string",
					Pattern: `^[\w_-.]$`,
				},
				Example: "Bearer someValidToken",
			},
			{
				Name:        "deviceId",
				In:          "path",
				Description: "Device Id to request data from.",
				Required:    true,
				Schema: &huma.Schema{
					Type:    "string",
					Pattern: `^\w{1,30}$`,
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

func (a *Api) GetDeviceData(ctx context.Context,
	in *struct {
		DeviceId string `path:"deviceId" pattern:"^[a-zA-Z0-9]{1,30}$" required:"true"`
		Body     datafetcher.DeviceDataRequest
	}) (*struct {
	Body *datafetcher.ConsolidatedDeviceData
}, error) {
	var relativeTime bool
	in.Body.Start = strings.TrimSpace(in.Body.Start)
	if RELATIVETIME_REGEX.MatchString(in.Body.Start) {
		relativeTime = true
	} else {
		if _, err := time.Parse(time.RFC3339Nano, in.Body.Start); err != nil {
			log.Printf("parsing start: %v", err)
			return nil, huma.Error400BadRequest("Start time is invalid rfc.")
		}
	}
	if relativeTime {
		in.Body.Stop = ""
	} else {
		if in.Body.Stop == "" {
			return nil, huma.Error400BadRequest(
				"Stop time is empty, when start is valid rfc format.")
		}
		in.Body.Stop = strings.TrimSpace(in.Body.Stop)
		if _, err := time.Parse(time.RFC3339Nano, in.Body.Stop); err != nil {
			log.Printf("parsing stop: %v")
			return nil, huma.Error400BadRequest("Stop time is invalid rfc.")
		}
	}
	//TODO: allow users to ask for multiple queryfields at once
	queryFields := []string{in.Body.QueryField}
	if in.Body.QueryField == "all" {
		attachedSensors, err := a.DeviceInfo.GetAttachedSensors(in.DeviceId)
		if err != nil {
			log.Printf("error getting attached sensors for: %s: %v", in.DeviceId, err)
			return nil, huma.Error500InternalServerError(
				"Internal error getting attached sensors for deviceId.")
		}
		queryFields, err = a.DeviceInfo.GetQueryFields(attachedSensors)
		if err != nil {
			log.Printf("error getting query fields for: %s: %v", in.DeviceId, err)
			return nil, huma.Error500InternalServerError(
				"Internal error getting query fields for deviceId.")
		}
	}
	user, ok := ctx.Value("user").(authstore.UserInfo)
	if !ok {
		log.Printf("user role is not UserInfo")
		return nil, huma.Error500InternalServerError(
			"Internal error getting user role.")
	}
	metadata := deviceinfo.DeviceInfo{
		Company:     user.Company,
		DeviceId:    in.DeviceId,
		Network:     user.Network,
		QueryFields: queryFields,
		Start:       in.Body.Start,
		Stop:        in.Body.Stop,
	}
	var err error
	if strings.ToLower(user.Role) == a.AdminRole {
		metadata.Company, err = a.DeviceInfo.GetCompany(in.DeviceId)
		if err != nil {
			log.Printf("error getting company for admin request on device: %s: %v",
				in.DeviceId, err)
			return nil, huma.Error500InternalServerError(
				"Internal error getting associated company for deviceId.")
		}
		metadata.Network, err = a.DeviceInfo.GetNetwork(in.DeviceId)
		if err != nil {
			log.Printf("error getting network for admin request on deviceId: %s: %v",
				in.DeviceId, err)
			return nil, huma.Error500InternalServerError(
				"Internal error getting associated network for deviceId.")
		}
	}
	deviceData, err := a.DataFetcher.GetData(metadata)
	if err != nil {
		log.Printf("error getting data: %v", err)
		return nil, huma.Error500InternalServerError(
			"Internal error fetching data.")
	}
	return &struct {
		Body *datafetcher.ConsolidatedDeviceData
	}{deviceData}, nil
}

func (a *Api) verifyToken(ctx huma.Context, next func(huma.Context)) {
	r, w := humamux.Unwrap(ctx)
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		log.Println("no auth header provided")
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	parts := strings.Split(authHeader, "Bearer")
	if len(parts) != 2 {
		log.Println("Invalid Authorization header format")
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	var tr tokenprovider.TokenResponse
	tr.Body = strings.TrimSpace(parts[1])
	tr.Body = strings.Trim(tr.Body, `"`)
	if !a.TokenProvider.IsValidToken(tr) {
		if err := a.AuthStore.DeleteToken(authstore.UserToken{Token: tr.Body}); err != nil {
			log.Printf("deleting token: %v", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError),
				http.StatusInternalServerError)
		}
		http.Error(w, http.StatusText(http.StatusUnauthorized),
			http.StatusUnauthorized)
		return
	}
	user, err := a.AuthStore.GetUser(tr.Body)
	if err != nil {
		log.Printf("getting user: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)
	}
	ctx = huma.WithValue(ctx, "user", user)
	next(ctx)
}

type LoginRequest struct {
	Auth string `header:"Authorization" doc:"Required in the format: Bearer base64(username:password)" required:"true"`
}

func (a *Api) Login(ctx context.Context,
	lr *LoginRequest) (*tokenprovider.TokenResponse, error) {
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
	if err = a.AuthStore.VerifyCredentials(username, password); err != nil {
		log.Printf("error: %v", err)
		return nil, huma.Error401Unauthorized("Bad credentials provided.")
	}
	token, expiry, err := a.TokenProvider.GenerateToken()
	if err != nil {
		return nil, huma.Error500InternalServerError(
			"Internal error generating an access token.")
	}
	ut := authstore.UserToken{
		Username:   username,
		Token:      token,
		Expiration: expiry,
	}
	if err = a.AuthStore.StoreToken(ut); err != nil {
		return nil, huma.Error500InternalServerError(
			"Internal error linking the token to the user.")
	}
	return &tokenprovider.TokenResponse{Body: token}, nil
}
