package main

import (
	"context"
	"log"
	"os"
	"regexp"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humamux"
	localhuma "github.com/datafarm-software/datafarm-api/api/huma"
	"github.com/datafarm-software/datafarm-api/datafetcher"
	deviceinfo "github.com/datafarm-software/datafarm-api/device-info"
	"github.com/datafarm-software/datafarm-api/tokenprovider"
	"github.com/gorilla/mux"
	"gopkg.in/yaml.v3"
)

var FileNameRegex = regexp.MustCompile(`^[\w-_/.]+\.{1,2}[\w]{1,5}$`)

func main() {
	args := os.Args
	filename := "api/openapi.yaml"
	if len(args) > 1 {
		if len(args) != 3 || args[1] != "--filename" {
			log.Fatalf("Usage: go run main.go --filename someName.yml")
		}
		filename = args[2]
		if !FileNameRegex.MatchString(filename) {
			log.Fatalf("Filename failed regex: %s", FileNameRegex)
		}
	}
	router := mux.NewRouter().PathPrefix("/api/v1").Subrouter()
	config := huma.DefaultConfig("SensorData API", "1.0.0")
	config.Servers = append(config.Servers, &huma.Server{URL: "/api/v1"})
	humaApi := humamux.New(router, config)
	localhuma.RegisterHumaOperations(humaApi,
		func(ctx huma.Context, next func(huma.Context)) {},
		func(ctx huma.Context, next func(huma.Context)) {},
		func(context.Context, *datafetcher.DeviceDataRequest) (
			*datafetcher.DeviceDataResponse, error) {
			return nil, nil
		},
		func(context.Context, *struct {
			Body []datafetcher.BatchDeviceDataRequest
		}) (
			*struct {
				Body datafetcher.BatchDeviceDataResponse
			}, error) {
			return nil, nil
		},
		func(context.Context, *tokenprovider.LoginRequest) (
			*tokenprovider.LoginResponse, error) {
			return nil, nil
		},
		func(context.Context, *deviceinfo.QueryFieldsRequest) (
			*deviceinfo.QueryFieldsResponse, error) {
			return nil, nil
		},
		func(context.Context, *deviceinfo.BatchQueryFieldsRequest) (
			*struct {
				Body deviceinfo.BatchQueryFieldsResponse
			}, error) {
			return nil, nil
		},
		func(context.Context, *struct{}) (*deviceinfo.DeviceIdsResponse, error) {
			return nil, nil
		},
	)
	doc := humaApi.OpenAPI()
	out, err := yaml.Marshal(doc)
	if err != nil {
		log.Fatalf("marshalling yaml: %v", err)
	}
	if err = os.WriteFile(filename, out, 0644); err != nil {
		log.Fatalf("writing yaml: %v", err)
	}
}
