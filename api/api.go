package api

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"slices"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humamux"
	"github.com/danielgtaylor/huma/v2/humacli"
	"github.com/datafarm-software/datafarm-api/api/authstore"
	localhuma "github.com/datafarm-software/datafarm-api/api/huma"
	"github.com/datafarm-software/datafarm-api/api/sensor/data"
	"github.com/datafarm-software/datafarm-api/api/sensor/info"
	"github.com/datafarm-software/datafarm-api/api/tokenprovider"
	"github.com/datafarm-software/telemetry"
	"github.com/datafarm-software/telemetry/metering"
	"github.com/datafarm-software/telemetry/middlewares"

	"github.com/datafarm-software/datafarm-api/api/redis"
	"github.com/gorilla/mux"
)

const EmptyPayloadLength int = 16

var pkgCtx context.Context

var QUERYFIELD_REGEX = regexp.MustCompile(`^[a-zA-Z0-9_\-\s:]*$`)

// var DEVICE_ID_REGEX = regexp.MustCompile(`\w{1,30}`)
var (
	RELATIVETIME_REGEX  = regexp.MustCompile(`-\d{1,3}(?:[hdw]|mo?)`)
	USERNAME_REGEX      = regexp.MustCompile(`^[\w .@]{1,75}`)
	UPPERCASE_REGEX     = regexp.MustCompile(`[A-Z]`)
	LOWERCASE_REGEX     = regexp.MustCompile(`[a-z]`)
	NUMBER_REGEX        = regexp.MustCompile(`[0-9]`)
	SPECIAL_CHARS_REGEX = regexp.MustCompile(`[@$!%*?&#]`)
)

type ApiOpts struct {
	RedisOpts      redis.RedisOpts `mapstructure:"Redis" validate:"required"`
	InfluxOpts     data.InfluxOpts `mapstructure:"Influx" validate:"required"`
	TelemetryOpts  telemetry.Opts  `mapstructure:"telemetry" validate:"required"`
	Port           string          `mapstructure:"port" validate:"required"`
	PrivateKeyFile string          `mapstructure:"privatekeyfile" validate:"required"`
	PublicKeyFile  string          `mapstructure:"publickeyfile" validate:"required"`
	Mode           localhuma.Mode  `mapstructure:"mode"`
}

type Api struct {
	DeviceInfo    info.Fetcher
	DataFetcher   data.Fetcher
	TokenProvider tokenprovider.TokenProvider
	AuthStore     authstore.AuthStore
	Port          string
	Telemetry     *telemetry.Telemetry
	mode          localhuma.Mode
}

func Start(opts ApiOpts) error {
	pkgCtx = context.Background()
	go func() {
		cleanupOldLimiters(pkgCtx)
	}()
	redis, err := redis.NewRedis(opts.RedisOpts)
	if err != nil {
		return err
	}
	df, err := data.NewInfluxDatafetcher(opts.InfluxOpts)
	if err != nil {
		return fmt.Errorf("error init influx: %v", err)
	}
	tokenAuth, err := tokenprovider.NewJwtAuth(os.DirFS("."), opts.PrivateKeyFile, opts.PublicKeyFile)
	if err != nil {
		return fmt.Errorf("error initializing jwt authstore: %v", err)
	}
	opts.TelemetryOpts.MeterOpts = append(opts.TelemetryOpts.MeterOpts, metering.WithLatency(),
		metering.WithMemoryUsage(), metering.WithActiveUsersCount(),
		metering.WithOtelRuntime(), metering.WithUptime(), metering.WithRequestCount())
	telem, err := telemetry.NewTelemetry(opts.TelemetryOpts)
	if err != nil {
		return fmt.Errorf("init telemetry: %v", err)
	}
	api := &Api{
		DeviceInfo:    redis,
		DataFetcher:   df,
		TokenProvider: tokenAuth,
		AuthStore:     redis,
		Port:          opts.Port,
		Telemetry:     telem,
		mode:          opts.Mode,
	}
	authstore.InitRoles()
	cli := humacli.New(func(hooks humacli.Hooks, _ *struct{}) {
		router, _ := api.SetupHumaRouter()
		server := &http.Server{
			Addr:    opts.Port,
			Handler: router,
		}
		hooks.OnStart(func() {
			log.Println("Server started on ", server.Addr)
			if err := server.ListenAndServe(); err != nil {
				if !errors.Is(err, http.ErrServerClosed) {
					log.Printf("HTTP server close: %v", err)
				}
			}
		})
		hooks.OnStop(func() {
			api.Close()
			server.Shutdown(pkgCtx)
		})
	})
	cli.Run()
	return nil
}

func (a *Api) SetupHumaRouter() (http.Handler, *huma.Config) {
	config := localhuma.Config(a.mode)
	router := mux.NewRouter()
	humaApi := humamux.New(router, config)
	mwProvider, _ := middlewares.NewHumaProvider(a.Telemetry)
	mws := mwProvider.TelemetryMiddlewares()
	mws = append(mws, a.VerifyToken)
	mws = slices.Insert(mws, 0, a.RateLimit)
	localhuma.SetupApiOperations(humaApi, a, mws)
	return router, &config
}

func (a *Api) Close() {
	var err error
	if err = a.DeviceInfo.Close(); err != nil {
		log.Printf("closing metadevice/data: %v", err)
	}
	if err = a.DataFetcher.Close(); err != nil {
		log.Printf("closing device/data: %v", err)
	}
	if err = a.TokenProvider.Close(); err != nil {
		log.Printf("closing token auth: %v", err)
	}
	if err = a.AuthStore.Close(); err != nil {
		log.Printf("closing basic auth: %v", err)
	}
	if err = a.Telemetry.Close(pkgCtx); err != nil {
		log.Printf("closing telemetry: %v", err)
	}
	log.Println("Api shutdown.")
}

func (a *Api) Mode() localhuma.Mode {
	return a.mode
}
