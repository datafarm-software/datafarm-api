package main

import (
	"bytes"
	_ "embed"
	"log"
	"os"
	"os/signal"
	"syscall"

	apiModule "github.com/geraud22/aquahaus-api/api"
	cfy "github.com/geraud22/config-from-yaml"
)

//go:embed config.yml
var config []byte

func main() {
	opts, err := cfy.LoadConfig[apiModule.ApiOpts](bytes.NewReader(config), "yaml", nil)
	if err != nil {
		log.Fatalf("error loading config: %v", err)
	}
	api, err := apiModule.NewApi(opts)
	if err != nil {
		log.Fatalf("error initializing app: %v", err)
	}
	api.StartHttpServer()
	log.Printf("Started server on %s", c.GetString("Port"))
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	<-sigChan
	log.Println("Received shutdown signal...")
	api.Shutdown()
	log.Println("Graceful shutdown complete.")
}
