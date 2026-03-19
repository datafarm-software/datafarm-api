package api

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humamux"
	"github.com/danielgtaylor/huma/v2/humacli"
	localhuma "github.com/datafarm-software/datafarm-api/api/huma"
	"github.com/datafarm-software/datafarm-api/authstore"
	"github.com/datafarm-software/datafarm-api/datafetcher"
	df "github.com/datafarm-software/datafarm-api/datafetcher"
	deviceinfo "github.com/datafarm-software/datafarm-api/device-info"
	"github.com/datafarm-software/datafarm-api/tokenprovider"

	"github.com/datafarm-software/datafarm-api/redis"
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
		AdminRole:     opts.AdminRole,
	}
	cli := humacli.New(func(hooks humacli.Hooks, options *ApiOpts) {
		router := mux.NewRouter().PathPrefix("/api/v1").Subrouter()
		config := huma.DefaultConfig("SensorData API", "1.0.0")
		config.Servers = append(config.Servers, &huma.Server{URL: "/api/v1"})
		humaApi := humamux.New(router, config)
		localhuma.RegisterHumaOperations(humaApi,
			api.VerifyToken, api.GetDeviceData, api.Login, api.GetQueryFields)
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

func (a *Api) GetDeviceData(ctx context.Context,
	in *localhuma.DeviceInput) (*localhuma.DeviceOutput, error) {
	in.Body.Start = strings.TrimSpace(in.Body.Start)
	if RELATIVETIME_REGEX.MatchString(in.Body.Start) {
		in.Body.Stop = ""
	} else {
		rfcStart, err := time.Parse(time.RFC3339Nano, in.Body.Start)
		if err != nil {
			log.Printf("parsing start: %v", err)
			return nil, huma.Error400BadRequest("Start time is invalid rfc.")
		}
		if rfcStart.UnixMilli() >= time.Now().UnixMilli() {
			return nil, huma.Error400BadRequest("Start time is in the future.")
		}
		if in.Body.Stop == "" {
			return nil, huma.Error400BadRequest("No stop time provided.")
		}
		in.Body.Stop = strings.TrimSpace(in.Body.Stop)
		rfcStop, err := time.Parse(time.RFC3339Nano, in.Body.Stop)
		if err != nil {
			return nil, huma.Error400BadRequest("Stop time is invalid rfc.")
		}
		if rfcStart.UnixMilli() >= rfcStop.UnixMilli() {
			return nil, huma.Error400BadRequest("Start time is greater than stop time.")
		}
	}
	if in.Body.QueryFields[0] == "all" {
		qf, err := a.DeviceInfo.GetQueryFields(in.DeviceId)
		if err != nil {
			log.Printf("error getting query fields for: %s: %v", in.DeviceId, err)
			return nil, huma.Error500InternalServerError(
				"Internal error getting query fields for deviceId.")
		}
		in.Body.QueryFields = qf.Body
	}
	deviceCompany, err := a.DeviceInfo.GetCompany(in.DeviceId)
	if err != nil {
		log.Printf("error getting company for admin request on device: %s: %v",
			in.DeviceId, err)
		return nil, huma.Error500InternalServerError(
			"Internal error getting associated company for deviceId.")
	}
	deviceNetwork, err := a.DeviceInfo.GetNetwork(in.DeviceId)
	if err != nil {
		log.Printf("error getting network for admin request on deviceId: %s: %v",
			in.DeviceId, err)
		return nil, huma.Error500InternalServerError(
			"Internal error getting associated network for deviceId.")
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
		QueryFields: in.Body.QueryFields,
		Start:       in.Body.Start,
		Stop:        in.Body.Stop,
	}
	isAdminUser := strings.ToLower(user.Role) == a.AdminRole
	if isAdminUser {
		metadata.Company = deviceCompany
		metadata.Network = deviceNetwork
	} else {
		if deviceCompany != user.Company {
			log.Printf("user: %s requested deviceId: %s. User Company: %s, Device Company: %s",
				user.Username, in.DeviceId, user.Company, deviceCompany)
			return nil, huma.Error401Unauthorized(
				"Unauthorized for access to this device.")
		}
		if deviceNetwork != user.Network {
			log.Printf("user: %s requested deviceId: %s. User Network: %s, Device Network: %s",
				user.Username, in.DeviceId, user.Network, deviceNetwork)
			return nil, huma.Error401Unauthorized(
				"Unauthorized for access to this device.")
		}
	}
	deviceData, err := a.DataFetcher.GetData(metadata)
	if err != nil {
		log.Printf("error getting data: %v", err)
		return nil, huma.Error500InternalServerError(
			"Internal error fetching data.")
	}
	return &localhuma.DeviceOutput{Body: deviceData}, nil
}

func (a *Api) VerifyToken(ctx huma.Context, next func(huma.Context)) {
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

func (a *Api) Login(ctx context.Context,
	lr *localhuma.LoginRequest) (*tokenprovider.TokenResponse, error) {
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

func (a *Api) GetQueryFields(ctx context.Context, in *localhuma.DeviceId) (
	*deviceinfo.QueryFields, error) {
	queryFields, err := a.DeviceInfo.GetQueryFields(in.DeviceId)
	if err != nil {
		return nil, huma.Error500InternalServerError(
			"Internal error while getting queryfields.")
	}
	return &queryFields, nil
}
