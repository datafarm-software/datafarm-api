package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"

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
var RELATIVETIME_REGEX = regexp.MustCompile(`-\d{1,3}(?:[hdw]|mo?)`)
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
			api.Close()
			server.Shutdown(ctx)
		})
	})
	cli.Run()
	return nil
}

func (a *Api) SetupHumaRouter() (http.Handler, *huma.Config) {
	config := huma.DefaultConfig("DataFarm SensorData API", "1.0.5")
	config.DocsPath = "/api/v1/docs"
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
	config.DefaultFormat = "application/json"
	config.Formats["text/csv"] = huma.Format{
		Marshal: func(w io.Writer, v any) error {
			cm, ok := v.(datafetcher.CsvMarshaller)
			if !ok {
				return fmt.Errorf("csv marshal did not receive marshaller, got: %T", v)
			}
			csv, _ := cm.Csv()
			_, err := w.Write([]byte(csv))
			return err
		},
		Unmarshal: func(data []byte, v any) error {
			return fmt.Errorf("text/csv request bodies are not supported")
		},
	}
	router := mux.NewRouter()
	humaApi := humamux.New(router, config)
	humaApi.OpenAPI().Servers = append(humaApi.OpenAPI().Servers, &huma.Server{
		URL: "/api/v1",
	})
	localhuma.RegisterHumaOperations(humaApi,
		a.RateLimit, a.VerifyToken,
		a.GetDeviceData, a.BatchGetDeviceData, a.Login, a.GetQueryFields,
		a.BatchGetQueryFields, a.GetDeviceIds, a.GetDeviceDataBoundary,
	)
	spec := humaApi.OpenAPI()
	op := spec.Paths["/batch/device/sensordata"].Post
	resp := op.Responses["200"]
	resp.Content["text/csv"] = &huma.MediaType{
		Schema: &huma.Schema{
			Description: "Clients are able to negotiate CSV formatted sensor data using the Accept header.",
			Type:        "string",
		},
	}
	op = spec.Paths["/device/{deviceId}/sensordata"].Get
	resp = op.Responses["200"]
	resp.Content["text/csv"] = &huma.MediaType{
		Schema: &huma.Schema{
			Description: "Clients are able to negotiate CSV formatted sensor data using the Accept header.",
			Type:        "string",
		},
	}
	return router, &config
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
