package datafetcher

import (
	"context"
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

func (i *InfluxDatafetcher) GetData(metadata apiModule.Metadata) (*apiModule.ConsolidatedDeviceData, error) {
	query := i.generateFluxQuery(metadata)
	result, err := i.queryApi.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("error querying influxdb: %v", err)
	}
	dataRows, err := i.extractValue(result)
	if err != nil {
		return nil, fmt.Errorf("error processing query result: %v", err)
	}
	return i.dataRows2ConsolidatedDeviceData(dataRows), nil
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

func (i *InfluxDatafetcher) extractValue(result *influxApi.QueryTableResult) ([]DataRow, error) {
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

func (i *InfluxDatafetcher) dataRows2ConsolidatedDeviceData(data []DataRow) *apiModule.ConsolidatedDeviceData {
	var deviceDataSlice []apiModule.DeviceData
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
			newDeviceData := apiModule.DeviceData{
				DeviceID:   row.DeviceID,
				Timestamp:  row.Time,
				SensorData: map[string]interface{}{row.Field: row.Value},
			}
			deviceDataSlice = append(deviceDataSlice, newDeviceData)
		}
	}
	return &apiModule.ConsolidatedDeviceData{
		DeviceData: deviceDataSlice,
	}
}

func (i *InfluxDatafetcher) FormatQueryRange(startTime, stopTime string) (interface{}, error) {
	relativeRange := false
	if startTime == "" {
		return nil, fmt.Errorf("no start time provided")
	}
	if _, err := time.Parse(time.RFC3339, startTime); err != nil {
		relativeRange = true
	} else {
		if stopTime == "" {
			return nil, fmt.Errorf("start time is rfc3339, but stop time is empty. cannot proceed.")
		}
	}
	if !relativeRange {
		if _, err := time.Parse(time.RFC3339, stopTime); err != nil {
			return nil, fmt.Errorf("Invalid RFC3339 stop timestamp: %v", err)
		}
		return fmt.Sprintf("start: %s, stop: %s", startTime, stopTime), nil
	}
	return fmt.Sprintf("start: %s", startTime), nil
}
