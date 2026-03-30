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
	"strconv"
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
	Port           string                 `mapstructure:"port" validate:"required"`
	PrivateKeyFile string                 `mapstructure:"privatekeyfile" validate:"required"`
	PublicKeyFile  string                 `mapstructure:"publickeyfile" validate:"required"`
}

type Api struct {
	Port          string
	DeviceInfo    deviceinfo.DeviceInfoFetcher
	DataFetcher   df.DataFetcher
	TokenProvider tokenprovider.TokenProvider
	AuthStore     authstore.AuthStore
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
	ctx = context.Background()
	go func() {
		cleanupOldLimiters(ctx)
	}()
	api := &Api{
		Port:       opts.Port,
		DeviceInfo: redis, DataFetcher: df,
		TokenProvider: tokenAuth,
		AuthStore:     redis,
	}
	authstore.InitRoles()
	cli := humacli.New(func(hooks humacli.Hooks, options *ApiOpts) {
		config := huma.DefaultConfig("SensorData API", "1.0.0")
		config.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
			"bearer": {
				Type:         "http",
				Scheme:       "bearer",
				Name:         "Authorization",
				In:           "header",
				BearerFormat: "JWT",
			},
			"basic": {
				Type:         "http",
				Scheme:       "basic",
				Name:         "Authorization",
				In:           "header",
				BearerFormat: "Basic",
			},
		}
		router := mux.NewRouter()
		humaApi := humamux.New(router, config)
		localhuma.RegisterHumaOperations(humaApi,
			api.RateLimit, api.VerifyToken,
			api.GetDeviceData, api.BatchGetDeviceData,
			api.Login, api.GetQueryFields, api.BatchGetQueryFields, api.GetDeviceIds,
			api.GetDeviceDataBoundary,
		)
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
			server.Shutdown(ctx)
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
	in *datafetcher.DeviceDataRequest) (*datafetcher.DeviceDataResponse, error) {
	in.Start = strings.TrimSpace(in.Start)
	if RELATIVETIME_REGEX.MatchString(in.Start) {
		older := CheckOlderThanNinetyDays(in.Start)
		if older {
			return nil, huma.Error400BadRequest("Relative start time older than 31 days.")
		}
		in.Stop = ""
	} else {
		rfcStart, err := time.Parse(time.RFC3339Nano, in.Start)
		if err != nil {
			log.Printf("parsing start: %v", err)
			return nil, huma.Error400BadRequest("Start time is invalid rfc.")
		}
		if rfcStart.UnixMilli() <= time.Now().Add(-31*24*time.Hour).UnixMilli() {
			return nil, huma.Error400BadRequest("Start time is greater than 31 days.")
		}
		if rfcStart.UnixMilli() >= time.Now().UnixMilli() {
			return nil, huma.Error400BadRequest("Start time is in the future.")
		}
		if in.Stop == "" {
			return nil, huma.Error400BadRequest("No stop time provided.")
		}
		in.Stop = strings.TrimSpace(in.Stop)
		rfcStop, err := time.Parse(time.RFC3339Nano, in.Stop)
		if err != nil {
			return nil, huma.Error400BadRequest("Stop time is invalid rfc.")
		}
		if rfcStart.UnixMilli() >= rfcStop.UnixMilli() {
			return nil, huma.Error400BadRequest("Start time is greater than stop time.")
		}
	}
	user, ok := ctx.Value("user").(authstore.UserInfo)
	if !ok {
		return nil, huma.Error500InternalServerError(
			"Internal error getting user.")
	}
	if in.QueryFields[0] == "all" {
		if !authstore.HasPermission(authstore.Role(user.Role),
			authstore.GetAllQueryFields) {
			return nil, huma.Error500InternalServerError(
				"Unauthorized for all queryfields.")
		}
		qf, err := a.DeviceInfo.GetQueryFields(in.DeviceId)
		if err != nil {
			log.Printf("error getting query fields for: %s: %v", in.DeviceId, err)
			return nil, huma.Error500InternalServerError(
				"Internal error getting query fields for deviceId.")
		}
		in.QueryFields = qf.QueryFields
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
	deviceInfo := deviceinfo.DeviceInfo{
		Company:     user.Company,
		DeviceId:    in.DeviceId,
		Network:     user.Network,
		QueryFields: in.QueryFields,
		Start:       in.Start,
		Stop:        in.Stop,
	}
	if deviceCompany != user.Company {
		if authstore.HasPermission(authstore.Role(user.Role), authstore.GetAnyCompany) {
			deviceInfo.Company = deviceCompany
		} else {
			log.Printf("user: %s requested deviceId: %s. User Company: %s, Device Company: %s",
				user.Username, in.DeviceId, user.Company, deviceCompany)
			return nil, huma.Error401Unauthorized(
				"Unauthorized access to this device.")
		}
	}
	if deviceNetwork != user.Network {
		if authstore.HasPermission(authstore.Role(user.Role), authstore.GetAnyNetwork) {
			deviceInfo.Network = deviceNetwork
		} else {
			log.Printf("user: %s requested deviceId: %s. User Network: %s, Device Network: %s",
				user.Username, in.DeviceId, user.Network, deviceNetwork)
			return nil, huma.Error401Unauthorized(
				"Unauthorized access to this device.")
		}
	}
	deviceData, err := a.DataFetcher.GetData(deviceInfo)
	if err != nil {
		log.Printf("error getting data: %v", err)
		return nil, huma.Error500InternalServerError(
			"Internal error fetching data.")
	}
	return &datafetcher.DeviceDataResponse{Body: deviceData}, nil
}

const MaxDays = 91
const MaxMinutes = 129600
const MaxSeconds = 7776000
const MaxHours = 2160
const MaxMonths = 3
const LowerCaseO = 0x6f
const Hyphen = 0x2d

func CheckOlderThanNinetyDays(start string) bool {
	if len(start) < 1 {
		return true
	}
	if start[0] != Hyphen {
		return true
	}
	start = strings.ReplaceAll(start, "-", "")
	var suffix string
	if start[len(start)-1] == byte(LowerCaseO) {
		suffix = "mo"
		start = strings.ReplaceAll(start, "mo", "")
	} else {
		suffix = string(start[len(start)-1])
		start = start[:len(start)-1]
	}
	number, err := strconv.Atoi(start)
	if err != nil {
		log.Printf("number conversion error: %v", err)
		return true
	}
	switch suffix {
	case "s":
		if number > MaxSeconds {
			return true
		}
	case "m":
		if number > MaxMinutes {
			return true
		}
	case "h":
		if number > MaxHours {
			return true
		}
	case "d":
		if number > MaxDays {
			return true
		}
	case "mo":
		if number > MaxMonths {
			return true
		}
	default:
		log.Printf("unknown suffix: %v", suffix)
		return true
	}
	return false
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
	var lr tokenprovider.LoginResponse
	lr.Body = strings.TrimSpace(parts[1])
	lr.Body = strings.Trim(lr.Body, `"`)
	if !a.TokenProvider.IsValidToken(lr) {
		if err := a.AuthStore.DeleteToken(authstore.UserToken{Token: lr.Body}); err != nil {
			log.Printf("deleting token: %v", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError),
				http.StatusInternalServerError)
		}
		http.Error(w, http.StatusText(http.StatusUnauthorized),
			http.StatusUnauthorized)
		return
	}
	user, err := a.AuthStore.GetUser(lr.Body)
	if err != nil {
		log.Printf("getting user: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)
	}
	ctx = huma.WithValue(ctx, "user", user)
	next(ctx)
}

func (a *Api) checkAccessToDevice(deviceId string, user authstore.UserInfo) (
	deviceinfo.DeviceInfo, int, error) {
	di := deviceinfo.DeviceInfo{DeviceId: deviceId}
	deviceCompany, err := a.DeviceInfo.GetCompany(deviceId)
	if err != nil {
		return di, http.StatusInternalServerError, fmt.Errorf(
			"Internal error checking device company.")
	}
	if user.Company != deviceCompany {
		if !authstore.HasPermission(authstore.Role(user.Role), authstore.GetAnyCompany) {
			return di, http.StatusUnauthorized, fmt.Errorf("Unauthorized access to this device.")
		}
	}
	deviceNetwork, err := a.DeviceInfo.GetNetwork(deviceId)
	if err != nil {
		return di, http.StatusInternalServerError, fmt.Errorf(
			"Internal error checking device network.")
	}
	if user.Network != deviceNetwork {
		if !authstore.HasPermission(authstore.Role(user.Role), authstore.GetAnyNetwork) {
			return di, http.StatusUnauthorized, fmt.Errorf("Unauthorized access to this device.")
		}
	}
	di.Company = deviceCompany
	di.Network = deviceNetwork
	return di, http.StatusOK, nil
}

func (a *Api) Login(ctx context.Context,
	ar *tokenprovider.LoginRequest) (*tokenprovider.LoginResponse, error) {
	parts := strings.Split(ar.Auth, " ")
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
	return &tokenprovider.LoginResponse{Body: token}, nil
}

func (a *Api) GetQueryFields(ctx context.Context, in *deviceinfo.QueryFieldsRequest) (
	*deviceinfo.QueryFieldsResponse, error) {
	user, ok := ctx.Value("user").(authstore.UserInfo)
	if !ok {
		return nil, huma.Error500InternalServerError(
			"Internal error getting user.")
	}
	_, code, err := a.checkAccessToDevice(in.DeviceId, user)
	if err != nil {
		switch code {
		case http.StatusUnauthorized:
			return nil, huma.Error401Unauthorized(
				"Unauthorized access to this device.")
		default:
			return nil, huma.Error500InternalServerError(
				"Internal error checking acess to DeviceId.")
		}
	}
	if !authstore.HasPermission(authstore.Role(user.Role),
		authstore.GetAllQueryFields) {
		return nil, huma.Error500InternalServerError("Access denied to QueryFields.")
	}
	queryFields, err := a.DeviceInfo.GetQueryFields(in.DeviceId)
	if err != nil {
		return nil, huma.Error500InternalServerError(
			"Internal error while getting queryfields.")
	}
	return &deviceinfo.QueryFieldsResponse{Body: queryFields}, nil
}

func (a *Api) BatchGetDeviceData(ctx context.Context,
	in *struct {
		Body []datafetcher.BatchDeviceDataRequest
	}) (*struct {
	Body datafetcher.BatchDeviceDataResponse
}, error) {
	var dr datafetcher.DeviceDataRequest
	var dataResp *datafetcher.DeviceDataResponse
	var deviceErr datafetcher.DeviceDataError
	var err error
	errSlice := make([]datafetcher.DeviceDataError, 0, len(in.Body))
	resultSlice := make([]datafetcher.DeviceData, 0, len(in.Body))
	for _, bdr := range in.Body {
		dr = datafetcher.DeviceDataRequest{
			DeviceId:    bdr.DeviceId,
			QueryFields: bdr.QueryFields,
			Start:       bdr.Start,
			Stop:        bdr.Stop,
		}
		dataResp, err = a.GetDeviceData(ctx, &dr)
		if err == nil {
			resultSlice = append(resultSlice, dataResp.Body...)
		} else {
			deviceErr.DeviceId = bdr.DeviceId
			deviceErr.Error = err.Error()
			errSlice = append(errSlice, deviceErr)
		}
	}
	return &struct {
		Body datafetcher.BatchDeviceDataResponse
	}{
		Body: datafetcher.BatchDeviceDataResponse{
			Results: resultSlice,
			Errors:  errSlice,
		},
	}, nil
}

func (a *Api) BatchGetQueryFields(ctx context.Context,
	in *deviceinfo.BatchQueryFieldsRequest) (*struct {
	Body deviceinfo.BatchQueryFieldsResponse
}, error) {
	var qr deviceinfo.QueryFieldsRequest
	var dataResp *deviceinfo.QueryFieldsResponse
	var deviceErr deviceinfo.QueryFieldsError
	var err error
	errSlice := make([]deviceinfo.QueryFieldsError, 0, len(in.Body))
	resultSlice := make([]deviceinfo.QueryFields, 0, len(in.Body))
	for _, deviceId := range in.Body {
		qr = deviceinfo.QueryFieldsRequest{
			DeviceId: deviceId,
		}
		dataResp, err = a.GetQueryFields(ctx, &qr)
		if err == nil {
			resultSlice = append(resultSlice, dataResp.Body)
		} else {
			deviceErr.DeviceId = deviceId
			deviceErr.Error = err.Error()
			errSlice = append(errSlice, deviceErr)
		}
	}
	return &struct {
		Body deviceinfo.BatchQueryFieldsResponse
	}{
		Body: deviceinfo.BatchQueryFieldsResponse{
			Results: resultSlice,
			Errors:  errSlice,
		},
	}, nil
}

func (a *Api) GetDeviceIds(ctx context.Context, _ *struct{}) (
	*deviceinfo.DeviceIdsResponse, error) {
	user, ok := ctx.Value("user").(authstore.UserInfo)
	if !ok {
		return nil, huma.Error500InternalServerError(
			"Internal error getting user.")
	}
	sr := deviceinfo.ScopeRestriction{
		Company: user.Company,
		Network: user.Network,
	}
	switch authstore.Role(user.Role) {
	case authstore.User:
		sr.Scope = deviceinfo.DevicesInCompanyInNetwork
	case authstore.NetworkUser:
		sr.Scope = deviceinfo.DevicesInNetwork
	case authstore.Admin:
		sr.Scope = deviceinfo.AllDevices
	default:
		return nil, huma.Error500InternalServerError(
			"Internal error determining devices scope.")
	}
	userDevices, err := a.DeviceInfo.GetDevices(sr)
	if err != nil {
		return nil, huma.Error500InternalServerError(
			"Internal error getting DeviceIds.")
	}
	return &deviceinfo.DeviceIdsResponse{
		Body: userDevices,
	}, nil
}

func (a *Api) GetDeviceDataBoundary(ctx context.Context, in *datafetcher.DataBoundaryRequest) (
	*datafetcher.DataBoundaryResponse, error) {
	user, ok := ctx.Value("user").(authstore.UserInfo)
	if !ok {
		return nil, huma.Error500InternalServerError(
			"Internal error getting user.")
	}
	di, code, err := a.checkAccessToDevice(in.DeviceId, user)
	if err != nil {
		switch code {
		case http.StatusUnauthorized:
			return nil, huma.Error401Unauthorized(
				"Unauthorized access to this device.")
		default:
			return nil, huma.Error500InternalServerError(
				"Internal error checking acess to DeviceId.")
		}
	}
	if !authstore.HasPermission(authstore.Role(user.Role),
		authstore.GetDataBoundary) {
		return nil, huma.Error500InternalServerError("Access denied to DataBoundary.")
	}
	dataBoundary, err := a.DataFetcher.GetDataBoundary(di)
	if err != nil {
		log.Printf("%s getting data boundary: %v", user.Username, err)
		return nil, huma.Error500InternalServerError(
			"Internal error getting DataBoundary.")
	}
	return &datafetcher.DataBoundaryResponse{Body: dataBoundary}, nil
}
