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

type DeviceId struct {
	DeviceId string `path:"deviceId" pattern:"^[a-zA-Z0-9]{1,30}$" required:"true"`
}

type DeviceInput struct {
	DeviceId string `path:"deviceId" pattern:"^[a-zA-Z0-9]{1,30}$" required:"true"`
	Body     datafetcher.DeviceDataRequest
}

type DeviceOutput struct {
	Body []datafetcher.DeviceData
}

type LoginRequest struct {
	Auth string `header:"Authorization" doc:"Required in the format: Bearer base64(username:password)" required:"true"`
}

func RegisterHumaOperations(api huma.API,
	verifyToken func(ctx huma.Context, next func(huma.Context)),
	getDeviceData HumaHandler[DeviceInput, DeviceOutput],
	login HumaHandler[LoginRequest, tokenprovider.TokenResponse],
	getQueryFields HumaHandler[DeviceId, deviceinfo.QueryFields],
) {
	registry := huma.NewMapRegistry("#/errors", huma.DefaultSchemaNamer)
	operation := huma.Operation{
		Method:      "GET",
		Path:        "/device/data/{deviceId}",
		Middlewares: huma.Middlewares{verifyToken},
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
	huma.Register(api, operation, getDeviceData)
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
	huma.Register(api, operation, login)
	operation = huma.Operation{
		Method:      "GET",
		Path:        "/device/queryfields/{deviceId}",
		Tags:        []string{"GET"},
		Middlewares: huma.Middlewares{verifyToken},
		Summary:     "Get all available QueryFields",
		Description: "Clients can use this route to get the device's QueryFields. A QueryField is defined as a metric which has data attached to it eg. A temperature sensor will have a 'temperature' QueryField.",
		RequestBody: &huma.RequestBody{},
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
							Detail: "Unknown username.",
						},
					},
				},
			},
		},
	}
	huma.Register(api, operation, getQueryFields)
}
