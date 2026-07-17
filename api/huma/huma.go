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

type HumaOperator interface {
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

func (h HumaError) Csv() (csvStr string, err error) {
	return fmt.Sprintf("Title, Status, Detail\n%s,%d,%s\n",
		h.Title, h.Status, h.Detail), nil
}

func RegisterHumaOperations(api huma.API, ho HumaOperator) {
	registry := huma.NewMapRegistry("#/errors", huma.DefaultSchemaNamer)
	operation := huma.Operation{
		Method:      "GET",
		Path:        "/device/{deviceId}/sensordata",
		Middlewares: huma.Middlewares{ho.RateLimit, ho.VerifyToken},
		Security: []map[string][]string{
			{"bearer": {}},
		},
		Parameters: []*huma.Param{
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
		Summary:     "Get Sensor Data",
		Description: "Clients can use this route to request data from a sensor using its device id.",
		RequestBody: &huma.RequestBody{},
		Responses: map[string]*huma.Response{
			"204": {
				Description: "No SensorData for the requested time period.",
			},
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
			"401": {
				Description: "Unauthorized",
				Content: map[string]*huma.MediaType{
					"application/json": {
						Schema: huma.SchemaFromType(registry, reflect.TypeFor[HumaError]()),
						Example: HumaError{
							Schema: "http://localhost:3030/schemas/ErrorModel.json",
							Title:  "Unauthorized",
							Status: http.StatusUnauthorized,
							Detail: "Unknown user.",
						},
					},
				},
			},
			"404": {
				Description: "Not Found",
				Content: map[string]*huma.MediaType{
					"application/json": {
						Schema: huma.SchemaFromType(registry, reflect.TypeFor[HumaError]()),
						Example: HumaError{
							Schema: "http://localhost:3030/schemas/ErrorModel.json",
							Title:  "Not Found",
							Status: http.StatusUnauthorized,
							Detail: "Device Not Found.",
						},
					},
				},
			},
		},
	}
	huma.Register(api, operation, ho.GetDeviceData)

	operation = huma.Operation{
		Method:      "POST",
		Path:        "/login",
		Tags:        []string{"POST"},
		Summary:     "Login",
		Description: "Clients can use this route to login and receive an active session token.",
		Security: []map[string][]string{
			{"basic": {}},
		},
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
							Detail: "Unknown user.",
						},
					},
				},
			},
		},
	}
	huma.Register(api, operation, ho.Login)

	operation = huma.Operation{
		Method:      "GET",
		Path:        "/device/{deviceId}/queryfields",
		Tags:        []string{"GET"},
		Middlewares: huma.Middlewares{ho.RateLimit, ho.VerifyToken},
		Summary:     "Get DeviceId QueryFields",
		Description: "Clients can use this route to get the device's QueryFields. A QueryField is defined as a metric which has data attached to it eg. A temperature sensor might have a 'temperature' QueryField.",
		RequestBody: &huma.RequestBody{},
		Security: []map[string][]string{
			{"bearer": {}},
		},
		Parameters: []*huma.Param{
			{
				Name:        "deviceId",
				In:          "path",
				Description: "Device Id to get QueryField information from.",
				Required:    true,
				Schema: &huma.Schema{
					Type:    "string",
					Pattern: `^\w{1,30}$`,
				},
			},
		},
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
							Detail: "Internal error getting queryFields.",
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
							Detail: "Unknown user.",
						},
					},
				},
			},
			"404": {
				Description: "Not Found",
				Content: map[string]*huma.MediaType{
					"application/json": {
						Schema: huma.SchemaFromType(registry, reflect.TypeFor[HumaError]()),
						Example: HumaError{
							Schema: "http://localhost:3030/schemas/ErrorModel.json",
							Title:  "Not Found",
							Status: http.StatusUnauthorized,
							Detail: "Device Not Found.",
						},
					},
				},
			},
		},
	}
	huma.Register(api, operation, ho.GetQueryFields)

	operation = huma.Operation{
		Method:      "POST",
		Path:        "/batch/device/sensordata",
		Middlewares: huma.Middlewares{ho.RateLimit, ho.VerifyToken},
		Security: []map[string][]string{
			{"bearer": {}},
		},
		Tags:        []string{"POST"},
		Summary:     "Batch Get Sensor Data",
		Description: "Clients can use this route to request data from multiple device ids.",
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
			"401": {
				Description: "Unauthorized",
				Content: map[string]*huma.MediaType{
					"application/json": {
						Schema: huma.SchemaFromType(registry, reflect.TypeFor[HumaError]()),
						Example: HumaError{
							Schema: "http://localhost:3030/schemas/ErrorModel.json",
							Title:  "Unauthorized",
							Status: http.StatusUnauthorized,
							Detail: "Unknown user.",
						},
					},
				},
			},
		},
	}
	huma.Register(api, operation, ho.BatchGetDeviceData)

	operation = huma.Operation{
		Method:      "POST",
		Path:        "/batch/device/queryfields",
		Middlewares: huma.Middlewares{ho.RateLimit, ho.VerifyToken},
		Security: []map[string][]string{
			{"bearer": {}},
		},
		Tags:        []string{"POST"},
		Summary:     "Batch Get DeviceId QueryFields",
		Description: "Clients can use this route to request QueryFields from multiple device ids.",
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
							Detail: "Internal error while getting queryfields for the device.",
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
							Detail: "Invalid DeviceId.",
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
							Detail: "Unknown user.",
						},
					},
				},
			},
			"404": {
				Description: "Not Found",
				Content: map[string]*huma.MediaType{
					"application/json": {
						Schema: huma.SchemaFromType(registry, reflect.TypeFor[HumaError]()),
						Example: HumaError{
							Schema: "http://localhost:3030/schemas/ErrorModel.json",
							Title:  "Not Found",
							Status: http.StatusUnauthorized,
							Detail: "Device Not Found.",
						},
					},
				},
			},
		},
	}
	huma.Register(api, operation, ho.BatchGetQueryFields)

	operation = huma.Operation{
		Method:      "GET",
		Path:        "/device/ids",
		Middlewares: huma.Middlewares{ho.RateLimit, ho.VerifyToken},
		Security: []map[string][]string{
			{"bearer": {}},
		},
		Tags:        []string{"GET"},
		Summary:     "Get Client DeviceIds",
		Description: "Clients can use this route to get the DeviceIds they have access to.",
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
							Detail: "Internal error while getting deviceids.",
						},
					}}},
			"401": {
				Description: "Unauthorized",
				Content: map[string]*huma.MediaType{
					"application/json": {
						Schema: huma.SchemaFromType(registry, reflect.TypeFor[HumaError]()),
						Example: HumaError{
							Schema: "http://localhost:3030/schemas/ErrorModel.json",
							Title:  "Unauthorized",
							Status: http.StatusUnauthorized,
							Detail: "Unknown user.",
						},
					},
				},
			},
		},
	}
	huma.Register(api, operation, ho.GetDeviceIds)

	operation = huma.Operation{
		Method:      "GET",
		Path:        "/device/{deviceId}/databoundary",
		Tags:        []string{"GET"},
		Middlewares: huma.Middlewares{ho.RateLimit, ho.VerifyToken},
		Summary:     "Get DeviceId DataBoundary",
		Description: "Clients can use this route to get the device's DataBoundary. A DataBoundary contains the oldest and most recent sensordata timestamps for the device.",
		RequestBody: &huma.RequestBody{},
		Security: []map[string][]string{
			{"bearer": {}},
		},
		Parameters: []*huma.Param{
			{
				Name:        "deviceId",
				In:          "path",
				Description: "Device Id to get DataBoundary information from.",
				Required:    true,
				Schema: &huma.Schema{
					Type:    "string",
					Pattern: `^\w{1,30}$`,
				},
			},
		},
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
							Detail: "Internal error getting DataBoundary.",
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
							Detail: "Unknown user.",
						},
					},
				},
			},
			"404": {
				Description: "Not Found",
				Content: map[string]*huma.MediaType{
					"application/json": {
						Schema: huma.SchemaFromType(registry, reflect.TypeFor[HumaError]()),
						Example: HumaError{
							Schema: "http://localhost:3030/schemas/ErrorModel.json",
							Title:  "Not Found",
							Status: http.StatusUnauthorized,
							Detail: "Device Not Found.",
						},
					},
				},
			},
		},
	}
	huma.Register(api, operation, ho.GetDeviceDataBoundary)
}

func Config() (config huma.Config) {
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
	config.DocsPath = "/v1/docs"
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
	humaApi.OpenAPI().Servers = append(humaApi.OpenAPI().Servers, &huma.Server{
		URL: "/v1",
	})
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
