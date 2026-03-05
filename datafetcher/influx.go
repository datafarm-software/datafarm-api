package datafetcher

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"os"
	"strings"
	"sync"
	"time"

	cfy "github.com/geraud22/config-from-yaml"
	deviceinfo "github.com/geraud22/datafarm-api/device-info"
	"github.com/geraud22/datafarm-api/metadatafetcher"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	influxApi "github.com/influxdata/influxdb-client-go/v2/api"
)

type DataRow struct {
	DeviceID string
	Field    string
	Time     time.Time
	Value    interface{}
}

type InfluxOpts struct {
	Org   string `mapstructure:"org" validate:"required"`
	Url   string `mapstructure:"url" validate:"required"`
	Token string `mapstructure:"token" validate:"required"`
}

type InfluxDatafetcher struct {
	db       influxdb2.Client
	queryApi influxApi.QueryAPI
}

func NewInfluxDatafetcher(opts InfluxOpts) (*InfluxDatafetcher, error) {
	db := influxdb2.NewClient(opts.Url, opts.Token)
	ok, err := db.Ping(context.Background())
	if err != nil {
		return nil, fmt.Errorf("influx ping error: %v", err)
	}
	if !ok {
		return nil, fmt.Errorf("influx server not running")
	}
	return &InfluxDatafetcher{
		db:       db,
		queryApi: db.QueryAPI(opts.Org),
	}, nil
}

func (i *InfluxDatafetcher) Close() error {
	i.db.Close()
	return nil
}

func (i *InfluxDatafetcher) GetData(metadata deviceinfo.DeviceInfo) (*ConsolidatedDeviceData, error) {
	formattedQueryRange, err := i.formatQueryRange(metadata.Start, metadata.Stop)
	if err != nil {
		return nil, err
	}
	query := i.generateFluxQuery(metadata, formattedQueryRange)
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

func (i *InfluxDatafetcher) generateFluxQuery(metadata metadatafetcher.Metadata, queryRange string) string {
	var queryBuilder strings.Builder
	queryBuilder.WriteString(fmt.Sprintf(`from(bucket: "%s")`, metadata.Network))
	queryBuilder.WriteString(fmt.Sprintf(` |> range(%s)`, queryRange))
	queryBuilder.WriteString(fmt.Sprintf(` |> filter(fn: (r) => r["_measurement"] == "%s")`, metadata.Company))
	queryBuilder.WriteString(fmt.Sprintf(` |> filter(fn: (r) => r["deviceID"] == "%s")`, metadata.DeviceId))
	queryBuilder.WriteString(` |> filter(fn: (r) => `)
	for _, filter := range metadata.QueryFields {
		queryBuilder.WriteString(fmt.Sprintf(` r["_field"] == "%s" or`, filter))
	}
	queryBuilder.WriteString(` false)`) //NOTE: for clean syntax query termination, after the last iteration's 'or'
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

func (i *InfluxDatafetcher) dataRows2ConsolidatedDeviceData(data []DataRow) *ConsolidatedDeviceData {
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

func (i *InfluxDatafetcher) formatQueryRange(startTime, stopTime string) (string, error) {
	relativeRange := false
	if startTime == "" {
		return "", fmt.Errorf("no start time provided")
	}
	if _, err := time.Parse(time.RFC3339, startTime); err != nil {
		relativeRange = true
	} else {
		if stopTime == "" {
			return "", fmt.Errorf("start time is rfc3339, but stop time is empty. cannot proceed.")
		}
	}
	if !relativeRange {
		if _, err := time.Parse(time.RFC3339, stopTime); err != nil {
			return "", fmt.Errorf("Invalid RFC3339 stop timestamp: %v", err)
		}
		return fmt.Sprintf("start: %s, stop: %s", startTime, stopTime), nil
	}
	return fmt.Sprintf("start: %s", startTime), nil
}

func (i *InfluxDatafetcher) PrepareDb(string, *ConsolidatedDeviceData) error {
	return nil
}

var testingInfluxOpts InfluxOpts
var once sync.Once

type TestingInflux struct {
	//NOTE: testMeasurement set during prepareDb
	testMeasurement string
	influx          *InfluxDatafetcher
}

func NewTestingInflux(configPath string) (DataFetcher, error) {
	var topErr error
	once.Do(func() {
		config, err := os.ReadFile(configPath)
		if err != nil {
			topErr = err
			return
		}
		opts, err := cfy.LoadConfig[struct {
			InfluxOpts InfluxOpts `mapstructure:"influx"`
		}](bytes.NewReader(config), "yaml", nil)
		if err != nil {
			topErr = err
			return
		}
		testingInfluxOpts = opts.InfluxOpts
	})
	if topErr != nil {
		return nil, topErr
	}
	db, err := NewInfluxDatafetcher(testingInfluxOpts)
	if err != nil {
		return nil, err
	}
	return &TestingInflux{
		influx: db,
	}, nil
}

func (t *TestingInflux) Close() error {
	deleteApi := t.influx.db.DeleteAPI()
	start := time.Now().Add(-168 * time.Hour)
	stop := time.Now()
	predicate := fmt.Sprintf(`_measurement="%s"`, t.testMeasurement)
	err := deleteApi.DeleteWithName(context.Background(), testingInfluxOpts.Org,
		testingInfluxOpts.Org, start, stop, predicate)
	if err != nil {
		return err
	}
	return t.influx.Close()
}

func (t *TestingInflux) PrepareDb(measurement string, mockDb *ConsolidatedDeviceData) error {
	if measurement == "" {
		return fmt.Errorf("measurement is empty")
	}
	t.testMeasurement = measurement
	writeApi := t.influx.db.WriteAPI(testingInfluxOpts.Org, testingInfluxOpts.Org)
	fields := make(map[string]any)
	for _, dd := range mockDb.DeviceData {
		for key, value := range dd.SensorData {
			_, isStringValue := value.(string)
			if isStringValue {
				continue
			}
			fields[key] = value
		}
		p := influxdb2.NewPoint(
			measurement,
			map[string]string{
				"deviceID": dd.DeviceID,
			},
			fields,
			dd.Timestamp,
		)
		writeApi.WritePoint(p)
	}
	writeApi.Flush()
	return nil
}

func (t *TestingInflux) GetData(metadata metadatafetcher.Metadata) (
	*ConsolidatedDeviceData, error) {
	return t.influx.GetData(metadata)
}
