package datafetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	apiModule "github.com/geraud22/aquahaus-api/api"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	influxApi "github.com/influxdata/influxdb-client-go/v2/api"
)

type InfluxDatafetcher struct {
	db       influxdb2.Client
	queryApi influxApi.QueryAPI
}

type ConsolidatedDeviceData struct {
	DeviceData []DeviceData `json:"payload"`
}

type DeviceData struct {
	DeviceID   string    `json:"rtuid"`
	Timestamp  time.Time `json:"timestamp"`
	SensorData map[string]interface{}
}

func NewInfluxDatafetcher(org, url, token string) (*InfluxDatafetcher, error) {
	db := influxdb2.NewClient(url, token)
	ok, err := db.Ping(context.Background())
	if err != nil {
		return nil, fmt.Errorf("influx ping error: %v", err)
	}
	if !ok {
		return nil, fmt.Errorf("influx server not running")
	}
	return &InfluxDatafetcher{
		db:       db,
		queryApi: db.QueryAPI(org),
	}, nil
}

func (i *InfluxDatafetcher) Close() error {
	i.db.Close()
	return nil
}

func (i *InfluxDatafetcher) GetData(metadata apiModule.Metadata) ([]byte, error) {
	query := i.generateFluxQuery(metadata)
	result, err := i.queryApi.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("error querying influxdb: %v", err)
	}
	deviceData, err := i.queryResultToConsolidatedDeviceData(result)
	if err != nil {
		return nil, fmt.Errorf("error converting query result to consolidated device data: %v", err)
	}
	jsonData, err := json.Marshal(deviceData)
	if err != nil {
		return nil, fmt.Errorf("error marshalling deviceData to json: %v", err)
	}
	return jsonData, nil
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

func (i *InfluxDatafetcher) queryResultToConsolidatedDeviceData(result *influxApi.QueryTableResult) (*ConsolidatedDeviceData, error) {
	deviceDataMap := make(map[string]map[time.Time]map[string]interface{})
	for result.Next() {
		timestamp := result.Record().Time()
		value := result.Record().Value()
		deviceID, ok := result.Record().ValueByKey("deviceID").(string)
		if !ok {
			return nil, fmt.Errorf("invalid deviceID format")
		}
		field := result.Record().Field()
		if _, exists := deviceDataMap[deviceID]; !exists {
			deviceDataMap[deviceID] = make(map[time.Time]map[string]interface{})
		}
		merged := false
		for existingTime := range deviceDataMap[deviceID] {
			if math.Abs(existingTime.Sub(timestamp).Seconds()) <= 10 {
				deviceDataMap[deviceID][existingTime][field] = value
				merged = true
				break
			}
		}
		if !merged {
			deviceDataMap[deviceID][timestamp] = map[string]interface{}{field: value}
		}
	}
	if err := result.Err(); err != nil {
		return nil, fmt.Errorf("query parsing error: %s", err)
	}
	consolidated := &ConsolidatedDeviceData{}
	for deviceID, timestamps := range deviceDataMap {
		for timestamp, sensorData := range timestamps {
			consolidated.DeviceData = append(consolidated.DeviceData, DeviceData{
				DeviceID:   deviceID,
				Timestamp:  timestamp,
				SensorData: sensorData,
			})
		}
	}
	if len(consolidated.DeviceData) < 2 {
		return nil, fmt.Errorf("no data found")
	}
	return consolidated, nil
}
