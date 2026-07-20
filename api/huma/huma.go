package huma

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/datafarm-software/datafarm-api/datafetcher"
	deviceinfo "github.com/datafarm-software/datafarm-api/device-info"
	"github.com/datafarm-software/datafarm-api/tokenprovider"
)

type Mode int

const (
	Production Mode = iota
	Development
)

const MajorApiVersionRoutePrefix = "/v1"

type HumaOperator interface {
	Mode() Mode
	RateLimit(ctx huma.Context, next func(huma.Context))
	VerifyToken(ctx huma.Context, next func(huma.Context))
	GetDeviceData(context.Context,
		*datafetcher.DeviceDataRequest) (*datafetcher.DeviceDataResponse, error)
	BatchGetDeviceData(context.Context, *struct {
		Body datafetcher.BatchDeviceDataRequest
	}) (*struct {
		Body *datafetcher.BatchDeviceDataResponse
	}, error)
	Login(context.Context, *tokenprovider.LoginRequest) (
		*tokenprovider.LoginResponse, error)
	GetQueryFields(context.Context, *deviceinfo.QueryFieldsRequest) (
		*deviceinfo.QueryFieldsResponse, error)
	BatchGetQueryFields(context.Context, *deviceinfo.BatchQueryFieldsRequest) (
		*struct {
			Body deviceinfo.BatchQueryFieldsResponse
		}, error)
	GetDeviceIds(context.Context, *struct{}) (*deviceinfo.DeviceIdsResponse, error)
	GetDeviceDataBoundary(context.Context, *datafetcher.DataBoundaryRequest) (
		*datafetcher.DataBoundaryResponse, error)
}

type HumaError struct {
	Schema string `json:"$schema" doc:"Link to Object Model."`
	Title  string `json:"title" doc:"Name associated with error code."`
	Status int    `json:"status" doc:"Http Status Code."`
	Detail string `json:"detail" doc:"Human Readable explanation of what went wrong."`
}

var HumaErrorRegistry huma.Registry = huma.NewMapRegistry("#/errors", huma.DefaultSchemaNamer)

func (h *HumaError) MediaType() *huma.MediaType {
	return &huma.MediaType{
		Schema:  huma.SchemaFromType(HumaErrorRegistry, reflect.TypeFor[HumaError]()),
		Example: h,
	}
}

func (h *HumaError) Csv() (csvStr string, err error) {
	return fmt.Sprintf("Title, Status, Detail\n%s,%d,%s\n",
		h.Title, h.Status, h.Detail), nil
}

func RegisterHumaOperations(api huma.API, ho HumaOperator) {
	operation := huma.Operation{RequestBody: &huma.RequestBody{}}
	Basic := []map[string][]string{{"basic": {}}}
	Bearer := []map[string][]string{{"bearer": {}}}
	FiveHundredExample := HumaError{
		Schema: "http://localhost:3030/schemas/ErrorModel.json",
		Title:  "Internal Server Error",
		Status: http.StatusInternalServerError,
	}
	FiveHundredResponse := &huma.Response{
		Description: "Internal Server Error",
		Content: map[string]*huma.MediaType{
			"application/json": FiveHundredExample.MediaType()}}
	FourHundredExample := HumaError{
		Schema: "http://localhost:3030/schemas/ErrorModel.json",
		Title:  "Bad Request",
		Status: http.StatusBadRequest,
		Detail: "No auth header provided.",
	}
	FourHundredResponse := &huma.Response{
		Description: "Bad Request",
		Content: map[string]*huma.MediaType{
			"application/json": FourHundredExample.MediaType()}}
	FourOhOneExample := HumaError{
		Schema: "http://localhost:3030/schemas/ErrorModel.json",
		Title:  "Unauthorized",
		Status: http.StatusUnauthorized,
		Detail: "Unknown user.",
	}
	FourOhOneResponse := &huma.Response{
		Description: "Unauthorized",
		Content: map[string]*huma.MediaType{
			"application/json": FourOhOneExample.MediaType()}}
	FourOhFourExample := HumaError{
		Schema: "http://localhost:3030/schemas/ErrorModel.json",
		Title:  "Not Found",
		Status: http.StatusUnauthorized,
		Detail: "Device Not Found.",
	}
	FourOhFourResponse := &huma.Response{
		Description: "Not Found",
		Content: map[string]*huma.MediaType{
			"application/json": FourOhFourExample.MediaType()}}
	operation.Responses = map[string]*huma.Response{
		"500": FiveHundredResponse,
		"400": FourHundredResponse,
		"401": FourOhOneResponse,
	}

	operation.Method = "POST"
	operation.Tags = []string{"POST"}

	operation.Path = "/login"
	operation.Summary = "Login"
	operation.Security = Basic
	operation.Description =
		"Clients can use this route to login and receive an active session token."
	FiveHundredExample.Detail = "Internal error logging in."
	operation.Responses["500"].Content["application/json"] = FiveHundredExample.MediaType()
	huma.Register(api, operation, ho.Login)

	operation.Security = Bearer
	operation.Middlewares = huma.Middlewares{ho.RateLimit, ho.VerifyToken}
	operation.Responses["404"] = FourOhFourResponse

	operation.Path = "/batch/device/sensordata"
	operation.Summary = "Batch Get Sensor Data"
	operation.Description = "Clients can use this route to request data from multiple device ids."
	FiveHundredExample.Detail = "Internal error while getting data for the device."
	operation.Responses["500"].Content["application/json"] = FiveHundredExample.MediaType()
	huma.Register(api, operation, ho.BatchGetDeviceData)

	operation.Path = "/batch/device/queryfields"
	operation.Summary = "Batch Get DeviceId QueryFields"
	operation.Description =
		"Clients can use this route to request QueryFields from multiple device ids."
	FiveHundredExample.Detail =
		"Internal error while getting queryfields for the device."
	operation.Responses["500"].Content["application/json"] = FiveHundredExample.MediaType()
	FourOhFourExample.Detail = "Device Not Found."
	operation.Responses["404"].Content["application/json"] = FourOhFourExample.MediaType()
	huma.Register(api, operation, ho.BatchGetQueryFields)

	deviceIdParam := &huma.Param{
		Name:     "deviceId",
		In:       "path",
		Required: true,
		Schema: &huma.Schema{
			Type:    "string",
			Pattern: `^\w{1,30}$`,
		},
	}

	operation.Method = "GET"
	operation.Tags = []string{"GET"}

	operation.Path = "/device/{deviceId}/sensordata"
	deviceIdParam.Description = "Device Id to request data from."
	operation.Parameters = []*huma.Param{deviceIdParam}
	operation.Summary = "Get Sensor Data"
	operation.Description =
		"Clients can use this route to request data from a sensor using its device id."
	operation.Responses["204"] = &huma.Response{
		Description: "No SensorData for the requested time period.",
	}
	FiveHundredExample.Detail = "Internal error while getting data for the device."
	operation.Responses["500"].Content["application/json"] = FiveHundredExample.MediaType()
	huma.Register(api, operation, ho.GetDeviceData)
	operation.Responses["204"] = nil
	operation.Parameters = []*huma.Param{}

	operation.Path = "/device/{deviceId}/queryfields"
	operation.Summary = "Get DeviceId QueryFields"
	operation.Description = "Clients can use this route to get the device's QueryFields. A QueryField is defined as a metric which has data attached to it eg. A temperature sensor might have a 'temperature' QueryField."
	deviceIdParam.Description = "Device Id to get QueryField information from."
	operation.Parameters = []*huma.Param{deviceIdParam}
	FiveHundredExample.Detail = "Internal error getting queryFields."
	operation.Responses["500"].Content["application/json"] = FiveHundredExample.MediaType()
	huma.Register(api, operation, ho.GetQueryFields)
	operation.Parameters = []*huma.Param{}

	operation.Path = "/device/ids"
	operation.Summary = "Get Client DeviceIds"
	operation.Description = "Clients can use this route to get the DeviceIds they have access to."
	FiveHundredExample.Detail = "Internal error while getting deviceids."
	operation.Responses["500"].Content["application/json"] = FiveHundredExample.MediaType()
	huma.Register(api, operation, ho.GetDeviceIds)

	operation.Path = "/device/{deviceId}/databoundary"
	operation.Summary = "Get DeviceId DataBoundary"
	operation.Description = "Clients can use this route to get the device's DataBoundary. A DataBoundary contains the oldest and most recent sensordata timestamps for the device."
	deviceIdParam.Description = "Device Id to get DataBoundary information from."
	operation.Parameters = []*huma.Param{deviceIdParam}
	FiveHundredExample.Detail = "Internal error getting DataBoundary."
	operation.Responses["500"].Content["application/json"] = FiveHundredExample.MediaType()
	huma.Register(api, operation, ho.GetDeviceDataBoundary)
	operation.Parameters = []*huma.Param{}
}

func Config(mode Mode) (config huma.Config) {
	config = huma.DefaultConfig("DataFarm SensorData API", "1.1.2")
	config.Info.Description = `
## Welcome

The DataFarm SensorData API provides our clients with access to their Sensor Data,
Device Metadata, and Export Functionality.

### Authentication

A Bearer token should first be obtained from the Login endpoint. All subsequent endpoints will require this for authentication.

### Content Types

All endpoints support JSON by default. Selected endpoints also support CSV by
setting request header:

Accept: text/csv

### Contributing

DataFarm welcomes external contribution to the API, through Open Source under the GPL-3.0 License. If you would like to contribute please visit the project's Github page to get started: www.github.com/datafarm-software/datafarm-api.
`
	var docsPath string
	if mode == Production {
		docsPath = MajorApiVersionRoutePrefix
	}
	config.DocsPath = docsPath + "/docs"
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
	defaultTransformer := huma.NewSchemaLinkTransformer("#/components/schemas/", "/schemas")
	config.CreateHooks = []func(huma.Config) huma.Config{
		func(c huma.Config) huma.Config {
			c.OnAddOperation = append(c.OnAddOperation, defaultTransformer.OnAddOperation)
			return c
		},
	}
	config.Transformers = []huma.Transformer{
		func(ctx huma.Context, status string, v any) (any, error) {
			negotiatedContentType := ctx.Header("Accept")
			if !strings.Contains(negotiatedContentType, "text/csv") {
				return defaultTransformer.Transform(ctx, status, v)
			}
			if errM, ok := v.(*huma.ErrorModel); ok {
				return HumaError{
					Title:  errM.Title,
					Status: errM.Status,
					Detail: errM.Detail,
				}, nil
			}
			return v, nil
		},
	}
	return
}

func SetupApi(humaApi huma.API, a HumaOperator) {
	if a.Mode() == Production {
		humaApi.OpenAPI().Servers = append(humaApi.OpenAPI().Servers, &huma.Server{
			URL: MajorApiVersionRoutePrefix,
		})
	}
	csvMediaType := &huma.MediaType{
		Schema: &huma.Schema{
			Description: "Clients are able to negotiate CSV formatted Sensor Data using the Accept header. Format of the CSV is dependent on the QueryFields associated with the DeviceId. Should there be any errors, clients can expect these to be included in the CSV.",
			Type:        "string",
		},
	}
	RegisterHumaOperations(humaApi, a)
	spec := humaApi.OpenAPI()
	op := spec.Paths["/batch/device/sensordata"].Post
	resp := op.Responses["200"]
	resp.Content["text/csv"] = csvMediaType
	op = spec.Paths["/device/{deviceId}/sensordata"].Get
	resp = op.Responses["200"]
	resp.Content["text/csv"] = csvMediaType
}
