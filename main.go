package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	apiModule "github.com/geraud22/aquahaus-api/api"
	"github.com/geraud22/aquahaus-api/authoriser"
	"github.com/geraud22/aquahaus-api/datafetcher"
	"github.com/geraud22/aquahaus-api/metadatafetcher"
	cfy "github.com/geraud22/config-from-yaml"
)

func main() {
	c := cfy.Get("config")
	metadataDB := c.GetInt("Redis.DB")
	authDB := c.GetInt("Redis.DB") + 2
	mf := metadatafetcher.NewRedisMetadata(c.GetString("Redis.Address"), c.GetString("Redis.Password"), metadataDB)
	df, err := datafetcher.NewInfluxDatafetcher(c.GetString("Influx.Org"), c.GetString("Influx.URL"), c.GetString("Influx.Token"))
	if err != nil {
		log.Fatalf("error initializing influxdata fetcher: %v", err)
	}
	tokenAuth, err := authoriser.NewJwtAuth(os.DirFS("."), c.GetString("PrivateKeyFile"), c.GetString("PublicKeyFile"))
	if err != nil {
		log.Fatalf("error initializing jwt authoriser: %v", err)
	}
	basicAuth := authoriser.NewRedisBasicAuth(c.GetString("Redis.Address"), c.GetString("Redis.Password"), authDB)
	opts, err := apiModule.NewApiOpts(c.GetString("Port"), mf, df, tokenAuth, basicAuth)
	if err != nil {
		log.Fatalf("error getting api opts: %v", err)
	}
	api, err := apiModule.NewApi(opts)
	if err != nil {
		log.Fatalf("error initializing app: %v", err)
	}
	api.StartHttpServer()
	log.Printf("Started server on %s", c.GetString("Port"))
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Received shutdown signal...")
	api.Shutdown()
	log.Println("Graceful shutdown complete.")
}
