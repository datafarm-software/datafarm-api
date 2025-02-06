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

type DataRow struct {
	DeviceID string
	Field    string
	Time     time.Time
	Value    interface{}
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
	dataRows, err := i.ExtractValue(result)
	if err != nil {
		return nil, fmt.Errorf("error converting query result to consolidated device data: %v", err)
	}
	deviceData := i.DataRows2ConsolidatedDeviceData(dataRows)
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

func (i *InfluxDatafetcher) ExtractValue(result *influxApi.QueryTableResult) ([]DataRow, error) {
	var records []DataRow
	for result.Next() {
		dataRow := DataRow{
			Time:     result.Record().Time(),
			Value:    result.Record().Value(),
			DeviceID: result.Record().ValueByKey("deviceID").(string),
			Field:    result.Record().Field(),
		}
		records = append(records, dataRow)
	}
	if result.Err() != nil {
		return nil, fmt.Errorf("query parsing error: %s", result.Err().Error())
	}
	return records, nil
}

func (i *InfluxDatafetcher) DataRows2ConsolidatedDeviceData(data []DataRow) *ConsolidatedDeviceData {
	var deviceDataSlice []DeviceData
	for _, row := range data {
		found := false
		for i, deviceData := range deviceDataSlice {
			if deviceData.DeviceID == row.DeviceID && math.Abs(float64(deviceData.Timestamp.Sub(row.Time).Seconds())) <= 10 {
				deviceDataSlice[i].SensorData[row.Field] = row.Value
				found = true
				break
			}
		}
		if !found {
			newDeviceData := DeviceData{
				DeviceID:   row.DeviceID,
				Timestamp:  row.Time,
				SensorData: map[string]interface{}{row.Field: row.Value},
			}
			deviceDataSlice = append(deviceDataSlice, newDeviceData)
		}
	}
	return &ConsolidatedDeviceData{
		DeviceData: deviceDataSlice,
	}
}
