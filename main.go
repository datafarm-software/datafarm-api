package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	apiModule "github.com/geraud22/aquahaus-api/api"
)

func main() {
	opts, err := apiModule.NewApiOpts(":8086")
	if err != nil {
		log.Fatalf("error getting api opts: %v", err)
	}
	api, err := apiModule.NewApi(opts)
	if err != nil {
		log.Fatalf("error initializing app: %v", err)
	}
	api.StartHttpServer()
	log.Println("Started server on :8086")
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Received shutdown signal...")
	api.Shutdown()
	log.Println("Graceful shutdown complete.")
}
