package datafetcher

import (
	"context"
	"fmt"
	"strings"

	apiModule "github.com/geraud22/aquahaus-api/api"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	influxApi "github.com/influxdata/influxdb-client-go/v2/api"
)

type InfluxDatafetcher struct {
	db       influxdb2.Client
	queryApi influxApi.QueryAPI
}

func NewInfluxDatafetcher(org, url, token string) (*InfluxDatafetcher, error) {
	db := influxdb2.NewClient(url, token)
	return &InfluxDatafetcher{
		db:       db,
		queryApi: db.QueryAPI(org),
	}, nil
}

func (i *InfluxDatafetcher) GetData(metadata apiModule.Metadata) ([]byte, error) {
	query := i.generateFluxQuery(metadata)
	result, err := i.queryApi.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("error querying influxdb: %v", err)
	}
	return nil, nil
}

func (i *InfluxDatafetcher) generateFluxQuery(metadata apiModule.Metadata) string {
	var queryBuilder strings.Builder
	queryBuilder.WriteString(fmt.Sprintf(`from(bucket: "%s")`, metadata.Network))
	queryBuilder.WriteString(fmt.Sprintf(` |> range(%s)`, metadata.QueryRange))
	queryBuilder.WriteString(fmt.Sprintf(` |> filter(fn: (r) => r["_measurement"] == "%s")`, metadata.Company))
	queryBuilder.WriteString(fmt.Sprintf(` |> filter(fn: (r) => r["deviceID"] == "%s")`, metadata.DeviceId))
	queryBuilder.WriteString(` |> filter(fn: (r) => `)
	for _, filter := range metadata.QueryFields {
		queryBuilder.WriteString(fmt.Sprintf(` r["_field"] == "%s" or`, filter))
	}
	queryBuilder.WriteString(` r["_field"] == "batv" or`)
	queryBuilder.WriteString(` r["_field"] == "BatV" or`)
	queryBuilder.WriteString(` r["_field"] == "signal_strength" or`)
	queryBuilder.WriteString(` r["_field"] == "rssi" or`)
	queryBuilder.WriteString(` r["_field"] == "snr")`)
	queryBuilder.WriteString(` |> yield(name: "last")`)
	return queryBuilder.String()
}
