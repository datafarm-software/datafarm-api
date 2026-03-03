package main

import (
	"bytes"
	_ "embed"
	"log"
	"os"

	cfy "github.com/geraud22/config-from-yaml"
	apiModule "github.com/geraud22/datafarm-api/api"
)

func main() {
	config, err := os.ReadFile("config.yml")
	if err != nil {
		log.Fatalf("error reading config file: %v", err)
	}
	opts, err := cfy.LoadConfig[apiModule.ApiOpts](bytes.NewReader(config), "yaml", nil)
	if err != nil {
		log.Fatalf("error loading config: %v", err)
	}
	if err := apiModule.Start(opts); err != nil {
		log.Fatalf("error initializing api: %v", err)
	}
}
