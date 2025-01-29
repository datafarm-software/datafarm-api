package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	apiModule "github.com/geraud22/aquahaus-api/api"
	"github.com/geraud22/aquahaus-api/datafetcher"
	"github.com/geraud22/aquahaus-api/metadatafetcher"
	cfy "github.com/geraud22/config-from-yaml"
)

func main() {
	c := cfy.Get("config")
	mf := metadatafetcher.NewRedisMetadata(c.GetString("Redis.Address"), c.GetString("Redis.Password"), c.GetInt("Redis.DB"))
	df, err := datafetcher.NewInfluxDatafetcher(c.GetString("Influx.Org"), c.GetString("Influx.URL"), c.GetString("Influx.Token"))
	if err != nil {
		log.Fatalf("error initializing influxdata fetcher: %v", err)
	}
	opts, err := apiModule.NewApiOpts(c.GetString("Port"), mf, df)
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
