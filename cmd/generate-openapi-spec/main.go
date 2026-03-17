package main

import (
	"log"
	"os"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humamux"
	"github.com/geraud22/datafarm-api/api"
	"github.com/gorilla/mux"
	"gopkg.in/yaml.v3"
)

func main() {
	api := &api.Api{}
	router := mux.NewRouter().PathPrefix("/api/v1").Subrouter()
	config := huma.DefaultConfig("SensorData API", "1.0.0")
	config.Servers = append(config.Servers, &huma.Server{URL: "/api/v1"})
	humaApi := humamux.New(router, config)
	api.RegisterHumaOperations(humaApi)
	doc := humaApi.OpenAPI()
	out, err := yaml.Marshal(doc)
	if err != nil {
		log.Fatalf("marshalling yaml: %v", err)
	}
	if err = os.WriteFile("api/openapi.yaml", out, 0644); err != nil {
		log.Fatalf("writing yaml: %v", err)
	}
}
