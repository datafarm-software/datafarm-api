package huma

import (
	"context"
	"net/http"
	"reflect"

	"github.com/danielgtaylor/huma/v2"
	"github.com/datafarm-software/datafarm-api/datafetcher"
	deviceinfo "github.com/datafarm-software/datafarm-api/device-info"
	"github.com/datafarm-software/datafarm-api/tokenprovider"
)

type HumaError struct {
	Schema string `json:"$schema" doc:"Link to Object Model."`
	Title  string `json:"title" doc:"Name associated with error code."`
	Status int    `json:"status" doc:"Http Status Code."`
	Detail string `json:"detail" doc:"Human Readable explanation of what went wrong."`
}

type HumaHandler[I, O any] func(context.Context, *I) (*O, error)

func RegisterHumaOperations(api huma.API,
	rateLimit func(ctx huma.Context, next func(huma.Context)),
	verifyToken func(ctx huma.Context, next func(huma.Context)),
	getSensorData HumaHandler[datafetcher.DeviceDataRequest, datafetcher.DeviceDataResponse],
	csvGetSensorData HumaHandler[datafetcher.DeviceDataRequest, struct {
		Body []byte `contentType:"text/csv"`
	}],
	batchGetDeviceData HumaHandler[
		struct {
			Body []datafetcher.BatchDeviceDataRequest
		},
		struct {
			Body datafetcher.BatchDeviceDataResponse
		}],
	login HumaHandler[tokenprovider.LoginRequest, tokenprovider.LoginResponse],
	getQueryFields HumaHandler[deviceinfo.QueryFieldsRequest, deviceinfo.QueryFieldsResponse],
	batchGetQueryFields HumaHandler[
		deviceinfo.BatchQueryFieldsRequest,
		struct {
			Body deviceinfo.BatchQueryFieldsResponse
		},
	],
	getDeviceIds HumaHandler[struct{}, deviceinfo.DeviceIdsResponse],
	getDeviceDataBoundary HumaHandler[datafetcher.DataBoundaryRequest, datafetcher.DataBoundaryResponse],
) {
	registry := huma.NewMapRegistry("#/errors", huma.DefaultSchemaNamer)

	operation := huma.Operation{
		Method:      "GET",
		Path:        "/device/{deviceId}/sensordata",
		Middlewares: huma.Middlewares{rateLimit, verifyToken},
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
	huma.Register(api, operation, getSensorData)

	operation = huma.Operation{
		Method:      "GET",
		Path:        "/device/{deviceId}/sensordata/csv",
		Middlewares: huma.Middlewares{rateLimit, verifyToken},
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
		Summary:     "Get CSV Sensor Data",
		Description: "Clients can use this route to request CSV formatted data from a sensor using its device id.",
		RequestBody: &huma.RequestBody{},
		Responses: map[string]*huma.Response{
			"200": {
				Description: "Format of the CSV file is dependent on the sensor's QueryFields.",
				Content: map[string]*huma.MediaType{
					"text/csv": {
						Schema: &huma.Schema{Type: "string", Format: "binary"},
					},
				},
			},
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
	huma.Register(api, operation, csvGetSensorData)

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
	huma.Register(api, operation, login)

	operation = huma.Operation{
		Method:      "GET",
		Path:        "/device/{deviceId}/queryfields",
		Tags:        []string{"GET"},
		Middlewares: huma.Middlewares{rateLimit, verifyToken},
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
	huma.Register(api, operation, getQueryFields)

	operation = huma.Operation{
		Method:      "POST",
		Path:        "/batch/device/sensordata",
		Middlewares: huma.Middlewares{rateLimit, verifyToken},
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
		},
	}
	huma.Register(api, operation, batchGetDeviceData)

	operation = huma.Operation{
		Method:      "POST",
		Path:        "/batch/device/queryfields",
		Middlewares: huma.Middlewares{rateLimit, verifyToken},
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
	huma.Register(api, operation, batchGetQueryFields)

	operation = huma.Operation{
		Method:      "GET",
		Path:        "/device/ids",
		Middlewares: huma.Middlewares{rateLimit, verifyToken},
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
	huma.Register(api, operation, getDeviceIds)

	operation = huma.Operation{
		Method:      "GET",
		Path:        "/device/{deviceId}/databoundary",
		Tags:        []string{"GET"},
		Middlewares: huma.Middlewares{rateLimit, verifyToken},
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
	huma.Register(api, operation, getDeviceDataBoundary)
}
