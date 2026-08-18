package api

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humamux"
	"github.com/danielgtaylor/huma/v2/humacli"
	"github.com/datafarm-software/datafarm-api/api/authstore"
	"github.com/datafarm-software/datafarm-api/api/datafetcher"
	deviceinfo "github.com/datafarm-software/datafarm-api/api/device-info"
	localhuma "github.com/datafarm-software/datafarm-api/api/huma"
	"github.com/datafarm-software/datafarm-api/api/telemetry"
	"github.com/datafarm-software/datafarm-api/api/tokenprovider"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"github.com/datafarm-software/datafarm-api/api/redis"
	"github.com/gorilla/mux"
)

const EmptyPayloadLength int = 16

var pkgCtx context.Context

var QUERYFIELD_REGEX = regexp.MustCompile(`^[a-zA-Z0-9_\-\s:]*$`)

// var DEVICE_ID_REGEX = regexp.MustCompile(`\w{1,30}`)
var RELATIVETIME_REGEX = regexp.MustCompile(`-\d{1,3}(?:[hdw]|mo?)`)
var USERNAME_REGEX = regexp.MustCompile(`^[\w .@]{1,75}`)
var UPPERCASE_REGEX = regexp.MustCompile(`[A-Z]`)
var LOWERCASE_REGEX = regexp.MustCompile(`[a-z]`)
var NUMBER_REGEX = regexp.MustCompile(`[0-9]`)
var SPECIAL_CHARS_REGEX = regexp.MustCompile(`[@$!%*?&#]`)

type ApiOpts struct {
	RedisOpts      redis.RedisOpts        `mapstructure:"Redis" validate:"required"`
	InfluxOpts     datafetcher.InfluxOpts `mapstructure:"Influx" validate:"required"`
	TelemetryOpts  telemetry.Opts         `mapstructure:"telemetry" validate:"required"`
	Port           string                 `mapstructure:"port" validate:"required"`
	PrivateKeyFile string                 `mapstructure:"privatekeyfile" validate:"required"`
	PublicKeyFile  string                 `mapstructure:"publickeyfile" validate:"required"`
	Mode           localhuma.Mode         `mapstructure:"mode"`
}

type Api struct {
	DeviceInfo    deviceinfo.DeviceInfoFetcher
	DataFetcher   datafetcher.DataFetcher
	TokenProvider tokenprovider.TokenProvider
	AuthStore     authstore.AuthStore
	Metric        *telemetry.OtlpRecorder
	Tracer        trace.Tracer
	Logger        *zap.Logger
	Port          string
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
	df, err := datafetcher.NewInfluxDatafetcher(opts.InfluxOpts)
	if err != nil {
		return fmt.Errorf("error init influx: %v", err)
	}
	tokenAuth, err := tokenprovider.NewJwtAuth(os.DirFS("."), opts.PrivateKeyFile, opts.PublicKeyFile)
	if err != nil {
		return fmt.Errorf("error initializing jwt authstore: %v", err)
	}
	res, err := resource.New(pkgCtx, resource.WithContainer())
	if err != nil {
		return fmt.Errorf("init resource: %v", err)
	}
	logger, loggerShutdown, err := telemetry.NewOtlpLogger(res)
	if err != nil {
		return fmt.Errorf("init logger: %v", err)
	}
	tracer, traceShutdown, err := telemetry.NewOtlpTracer(res)
	if err != nil {
		return fmt.Errorf("init tracer: %v", err)
	}
	meter, meterShutdown, err := telemetry.NewOtlpRecorder(res, opts.TelemetryOpts.MeterEndpoint)
	if err != nil {
		return fmt.Errorf("init meter: %v", err)
	}
	api := &Api{
		DeviceInfo:    redis,
		DataFetcher:   df,
		TokenProvider: tokenAuth,
		AuthStore:     redis,
		Port:          opts.Port,
		mode:          opts.Mode,
		Logger:        logger,
		Tracer:        tracer,
		Metric:        meter,
	}
	authstore.InitRoles()
	cli := humacli.New(func(hooks humacli.Hooks, options *ApiOpts) {
		router, _ := api.SetupHumaRouter()
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
			api.Close([]telemetry.Shutdown{
				loggerShutdown, traceShutdown, meterShutdown,
			}...)
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
	localhuma.SetupApi(humaApi, a)
	return router, &config
}

func (a *Api) Close(shutdownFuncs ...telemetry.Shutdown) {
	var err error
	if err = a.DeviceInfo.Close(); err != nil {
		log.Printf("error closing metadatafetcher: %v", err)
	}
	if err = a.DataFetcher.Close(); err != nil {
		log.Printf("error closing datafetcher: %v", err)
	}
	if err = a.TokenProvider.Close(); err != nil {
		log.Printf("error closing token auth: %v", err)
	}
	if err = a.AuthStore.Close(); err != nil {
		log.Printf("error closing basic auth: %v", err)
	}
	for _, s := range shutdownFuncs {
		err = errors.Join(err, s(pkgCtx))
	}
	if err != nil {
		log.Printf("telemetry shutdown funcs: %v", err)
	}
	log.Println("Api shutdown.")
}

func (a *Api) Mode() localhuma.Mode {
	return a.mode
}
