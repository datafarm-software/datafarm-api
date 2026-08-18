package huma

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/datafarm-software/datafarm-api/api/datafetcher"
	deviceinfo "github.com/datafarm-software/datafarm-api/api/device-info"
	"github.com/datafarm-software/datafarm-api/api/tokenprovider"
)

type Mode int

const (
	Production Mode = iota
	Development
)

const MajorApiVersionRoutePrefix = "/v1"

type HumaOperator interface {
	Mode() Mode
	LogRequest(ctx huma.Context, next func(huma.Context))
	TraceRequest(ctx huma.Context, next func(huma.Context))
	CountApiRequest(ctx huma.Context, next func(huma.Context))
	RecordLatency(ctx huma.Context, next func(huma.Context))
	RateLimit(ctx huma.Context, next func(huma.Context))
	VerifyToken(ctx huma.Context, next func(huma.Context))
	GetSensorData(context.Context,
		*datafetcher.SensorDataRequest) (*datafetcher.SensorDataResponse, error)
	BatchGetSensorData(context.Context, *struct {
		Body datafetcher.BatchSensorDataRequest
	}) (*struct {
		Body *datafetcher.BatchSensorDataResponse
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
	GetSensorDataBoundary(context.Context, *datafetcher.DataBoundaryRequest) (
		*datafetcher.DataBoundaryResponse, error)
	BatchGetSensorDataBoundary(context.Context,
		*struct {
			Body datafetcher.BatchDataBoundaryRequest
		}) (
		*struct {
			Body datafetcher.BatchDataBoundaryResponse
		}, error)
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

func (h HumaError) Csv() (csvStr string, err error) {
	return fmt.Sprintf("Title, Status, Detail\n%s,%d,%s\n",
		h.Title, h.Status, h.Detail), nil
}

func FiveHundredExample() *HumaError {
	return &HumaError{
		Schema: "http://localhost:3030/schemas/ErrorModel.json",
		Title:  "Internal Server Error",
		Status: http.StatusInternalServerError,
	}
}

func FiveHundredResponse() *huma.Response {
	return &huma.Response{
		Description: "Internal Server Error",
		Content: map[string]*huma.MediaType{
			"application/json": FiveHundredExample().MediaType()}}
}

func FourHundredExample() *HumaError {
	return &HumaError{
		Schema: "http://localhost:3030/schemas/ErrorModel.json",
		Title:  "Bad Request",
		Status: http.StatusBadRequest,
		Detail: "No Authorization Header Provided.",
	}
}

func FourHundredResponse() *huma.Response {
	return &huma.Response{
		Description: "Bad Request",
		Content: map[string]*huma.MediaType{
			"application/json": FourHundredExample().MediaType()}}
}

func FourOhOneExample() *HumaError {
	return &HumaError{
		Schema: "http://localhost:3030/schemas/ErrorModel.json",
		Title:  "Unauthorized",
		Status: http.StatusUnauthorized,
		Detail: "Unknown user.",
	}
}

func FourOhOneResponse() *huma.Response {
	return &huma.Response{
		Description: "Unauthorized",
		Content: map[string]*huma.MediaType{
			"application/json": FourOhOneExample().MediaType()}}
}

func FourOhFourExample() *HumaError {
	return &HumaError{
		Schema: "http://localhost:3030/schemas/ErrorModel.json",
		Title:  "Not Found",
		Status: http.StatusUnauthorized,
		Detail: "Device Not Found.",
	}
}

func FourOhFourResponse() *huma.Response {
	return &huma.Response{
		Description: "Not Found",
		Content: map[string]*huma.MediaType{
			"application/json": FourOhFourExample().MediaType()}}
}

var Basic = []map[string][]string{{"basic": {}}}
var Bearer = []map[string][]string{{"bearer": {}}}

func baseOperation(method string, middlewares *huma.Middlewares) huma.Operation {
	op := huma.Operation{
		RequestBody: &huma.RequestBody{},
		Responses: map[string]*huma.Response{
			"500": FiveHundredResponse(),
			"400": FourHundredResponse(),
			"401": FourOhOneResponse(),
			"404": FourOhFourResponse(),
		},
		Method:   method,
		Tags:     []string{method},
		Security: Bearer,
	}
	if middlewares != nil {
		op.Middlewares = *middlewares
	}
	return op
}

func RegisterHumaOperations(api huma.API, ho HumaOperator) {
	mw := []func(ctx huma.Context, next func(huma.Context)){
		ho.RateLimit, ho.CountApiRequest, ho.TraceRequest, ho.LogRequest, ho.RecordLatency,
		ho.VerifyToken,
	}
	noTokenVerification := huma.Middlewares(mw[:len(mw)-1])
	allMw := huma.Middlewares(mw)
	op := baseOperation("POST", &noTokenVerification)
	op.Path = "/login"
	op.Summary = "Login"
	op.Security = Basic
	op.Description =
		"Clients can use this route to login and receive an active session token."
	fh := FiveHundredExample()
	fh.Detail = "Internal error logging in."
	op.Responses["500"].Content["application/json"] = fh.MediaType()
	op.Responses["404"] = &huma.Response{}
	huma.Register(api, op, ho.Login)

	op = baseOperation("POST", &allMw)
	op.Path = "/batch/device/sensordata"
	op.Summary = "Batch Get Sensor Data"
	op.Description = "Clients can use this route to request data from multiple device ids."
	op.Responses["500"] = &huma.Response{}
	op.Responses["404"] = &huma.Response{}
	huma.Register(api, op, ho.BatchGetSensorData)

	op = baseOperation("POST", &allMw)
	op.Path = "/batch/device/queryfields"
	op.Summary = "Batch Get DeviceId QueryFields"
	op.Description =
		"Clients can use this route to request QueryFields from multiple device ids."
	op.Responses["500"] = &huma.Response{}
	op.Responses["404"] = &huma.Response{}
	huma.Register(api, op, ho.BatchGetQueryFields)

	op = baseOperation("POST", &allMw)
	op.Path = "/batch/device/databoundary"
	op.Summary = "Batch Get DeviceId DataBoundary"
	op.Description = "Clients can use this route to get the DataBoundary of multiple devices."
	op.Responses["500"] = &huma.Response{}
	op.Responses["404"] = &huma.Response{}
	huma.Register(api, op, ho.BatchGetSensorDataBoundary)

	deviceIdParam := &huma.Param{
		Name:     "deviceId",
		In:       "path",
		Required: true,
		Schema: &huma.Schema{
			Type:    "string",
			Pattern: `^\w{1,30}$`,
		},
	}

	op = baseOperation("GET", &allMw)
	op.Path = "/device/{deviceId}/sensordata"
	deviceIdParam.Description = "Device Id to request data from."
	op.Parameters = []*huma.Param{deviceIdParam}
	op.Summary = "Get Sensor Data"
	op.Description =
		"Clients can use this route to request data from a sensor using its device id."
	op.Responses["204"] = &huma.Response{
		Description: "No SensorData for the requested time period.",
	}
	fh = FiveHundredExample()
	fh.Detail = "Internal error while getting data for the device."
	op.Responses["500"].Content["application/json"] = fh.MediaType()
	huma.Register(api, op, ho.GetSensorData)
	op.Responses["204"] = &huma.Response{}
	op.Parameters = []*huma.Param{}

	op = baseOperation("GET", &allMw)
	op.Path = "/device/{deviceId}/queryfields"
	op.Summary = "Get DeviceId QueryFields"
	op.Description = "Clients can use this route to get the device's QueryFields. A QueryField is defined as a metric which has data attached to it eg. A temperature sensor might have a 'temperature' QueryField."
	deviceIdParam.Description = "Device Id to get QueryField information from."
	op.Parameters = []*huma.Param{deviceIdParam}
	fh = FiveHundredExample()
	fh.Detail = "Internal error getting queryFields."
	op.Responses["500"].Content["application/json"] = fh.MediaType()
	huma.Register(api, op, ho.GetQueryFields)

	op = baseOperation("GET", &allMw)
	op.Path = "/device/ids"
	op.Summary = "Get Client DeviceIds"
	op.Description = "Clients can use this route to get the DeviceIds they have access to."
	fh = FiveHundredExample()
	fh.Detail = "Internal error while getting deviceids."
	op.Responses["500"].Content["application/json"] = fh.MediaType()
	op.Responses["404"] = &huma.Response{}
	huma.Register(api, op, ho.GetDeviceIds)

	op = baseOperation("GET", &allMw)
	op.Path = "/device/{deviceId}/databoundary"
	op.Summary = "Get DeviceId DataBoundary"
	op.Description = "Clients can use this route to get the device's DataBoundary. A DataBoundary contains the oldest and most recent sensordata timestamps for the device."
	deviceIdParam.Description = "Device Id to get DataBoundary information from."
	op.Parameters = []*huma.Param{deviceIdParam}
	fh = FiveHundredExample()
	fh.Detail = "Internal error getting DataBoundary."
	op.Responses["500"].Content["application/json"] = fh.MediaType()
	huma.Register(api, op, ho.GetSensorDataBoundary)
}

func Config(mode Mode) (config huma.Config) {
	config = huma.DefaultConfig("DataFarm SensorData API", "1.1.4")
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
			Description: "Clients are able to negotiate CSV formatted Sensor Data using the Accept header. Format of the CSV is dependent on the QueryFields associated with the DeviceId. Timestamps will be in UTC timezone and RFC3339 Format. Should there be any errors, clients can expect these to be included in the CSV.",
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
