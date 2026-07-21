package datafetcher

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	deviceinfo "github.com/datafarm-software/datafarm-api/device-info"
	cfy "github.com/geraud22/config-from-yaml"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	influxApi "github.com/influxdata/influxdb-client-go/v2/api"
)

const TestOrg = "test-org"

var ctx context.Context

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

func (i *InfluxDatafetcher) GetData(metadata deviceinfo.DeviceInfo) (
	DeviceDataSlice, error) {
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
	dd, err := i.dataRows2DeviceData(dataRows)
	if err != nil {
		return nil, err
	}
	return dd, nil
}

func (i *InfluxDatafetcher) generateFluxQuery(metadata deviceinfo.DeviceInfo, queryRange string) string {
	var queryBuilder strings.Builder
	fmt.Fprintf(&queryBuilder, `from(bucket: "%s")`, metadata.Network)
	fmt.Fprintf(&queryBuilder, ` |> range(%s)`, queryRange)
	fmt.Fprintf(&queryBuilder, ` |> filter(fn: (r) => r["_measurement"] == "%s")`,
		metadata.Company)
	fmt.Fprintf(&queryBuilder, ` |> filter(fn: (r) => r["deviceID"] == "%s")`,
		metadata.DeviceId)
	fmt.Fprintf(&queryBuilder, ` |> filter(fn: (r) => `)
	for _, filter := range metadata.QueryFields {
		fmt.Fprintf(&queryBuilder, ` r["_field"] == "%s" or`, filter)
	}
	//NOTE: for clean syntax query termination, after the last iteration's 'or'
	fmt.Fprintf(&queryBuilder, ` false)`)
	fmt.Fprintf(&queryBuilder, ` |> yield(name: "last")`)
	return queryBuilder.String()
}

func (i *InfluxDatafetcher) extractValue(result *influxApi.QueryTableResult) ([]DataRow, error) {
	var records []DataRow
	for result.Next() {
		dataRow := DataRow{
			Time:     result.Record().Time().Local(),
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

func (i *InfluxDatafetcher) dataRows2DeviceData(data []DataRow) ([]DeviceData, error) {
	var deviceDataSlice []DeviceData
	var found bool
	var value float64
	var timestampWithin10Seconds bool
	for _, row := range data {
		found = false
		switch v := row.Value.(type) {
		case float64:
			value = v
		case int64:
			value = float64(v)
		case int:
			value = float64(v)
		default:
			value = float64(0)
		}
		//TODO: 10 seconds seems too long, check this
		for i, deviceData := range deviceDataSlice {
			timestampWithin10Seconds = math.Abs(float64(
				deviceData.Timestamp.Sub(row.Time).Seconds())) <= 10
			if deviceData.DeviceID == row.DeviceID &&
				timestampWithin10Seconds {
				deviceDataSlice[i].SensorData[row.Field] = value
				found = true
				break
			}
		}
		if !found {
			newDeviceData := DeviceData{
				DeviceID:   row.DeviceID,
				Timestamp:  row.Time,
				SensorData: map[string]float64{row.Field: value},
			}
			deviceDataSlice = append(deviceDataSlice, newDeviceData)
		}
	}
	return deviceDataSlice, nil
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

func (i *InfluxDatafetcher) GetDataBoundary(deviceInfo deviceinfo.DeviceInfo) (
	DataBoundary, error) {
	dataBoundary := DataBoundary{DeviceId: deviceInfo.DeviceId}
	queryBuilder := strings.Builder{}
	fmt.Fprintf(&queryBuilder, `data = from(bucket: "%s") `, deviceInfo.Network)
	fmt.Fprintf(&queryBuilder, "|> range(start: 0) ")
	fmt.Fprintf(&queryBuilder, `
		|> filter(fn: (r) => r._measurement == "%s" and r.deviceID == "%s" and r._field == "batv") `,
		deviceInfo.Company, deviceInfo.DeviceId)
	fmt.Fprintf(&queryBuilder, `|> group() `)
	fmt.Fprintf(&queryBuilder, `
		firstTime = data |> first()
		lastTime = data |> last()
		union(tables: [firstTime, lastTime])
		  |> sort(columns: ["_time"], desc: false)`)
	result, err := i.queryApi.Query(context.Background(), queryBuilder.String())
	if err != nil {
		return dataBoundary, fmt.Errorf("error querying influxdb: %v", err)
	}
	dataRows, err := i.extractValue(result)
	if err != nil {
		return dataBoundary, fmt.Errorf("error processing query result: %v", err)
	}
	if len(dataRows) != 2 {
		return dataBoundary, fmt.Errorf("dataBoundary dataRows returned is not 2, instead: %d", len(dataRows))
	}
	dataBoundary.Start = dataRows[0].Time
	dataBoundary.Stop = dataRows[1].Time
	return dataBoundary, nil
}

func (i *InfluxDatafetcher) PrepareDb(*deviceinfo.Schema, DeviceDataSlice) error {
	return nil
}

var testingInfluxOpts InfluxOpts
var once sync.Once

type TestingInflux struct {
	influx *InfluxDatafetcher
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
	testingInfluxOpts.Org = TestOrg
	db, err := NewInfluxDatafetcher(testingInfluxOpts)
	if err != nil {
		return nil, err
	}
	ctx = context.Background()
	return &TestingInflux{
		influx: db,
	}, nil
}

func (t *TestingInflux) Close() error {
	orgApi := t.influx.db.OrganizationsAPI()
	org, err := orgApi.FindOrganizationByName(ctx, TestOrg)
	if err != nil {
		return fmt.Errorf("finding org: %v", err)
	}
	if org == nil {
		return fmt.Errorf("returned org is nil")
	}
	if err = orgApi.DeleteOrganization(ctx, org); err != nil {
		return fmt.Errorf("deleting org: %v", err)
	}
	return t.influx.Close()
}

func (t *TestingInflux) PrepareDb(allDevicesInfo *deviceinfo.Schema, deviceData DeviceDataSlice) error {
	if allDevicesInfo == nil {
		return nil
	}
	if deviceData == nil {
		return nil
	}
	deviceInfoMap := deviceInfoMap(allDevicesInfo)
	orgApi := t.influx.db.OrganizationsAPI()
	org, err := orgApi.CreateOrganizationWithName(ctx, TestOrg)
	if err != nil {
		return fmt.Errorf("org api: %v", err)
	}
	bucketsApi := t.influx.db.BucketsAPI()
	uniqueNetworks := make([]string, 0, len(allDevicesInfo.DeviceNetworks))
	for _, dd := range allDevicesInfo.DeviceNetworks {
		if slices.Contains(uniqueNetworks, dd.Network) {
			continue
		}
		uniqueNetworks = append(uniqueNetworks, dd.Network)
		if _, err = bucketsApi.CreateBucketWithName(ctx, org, dd.Network); err != nil {
			if strings.Contains(err.Error(), "exists") {
				err = nil
			} else {
				break
			}
		}
	}
	if err != nil {
		return fmt.Errorf("buckets api: %v", err)
	}
	var writeApi influxApi.WriteAPI
	var ok bool
	var deviceInfo deviceinfo.DeviceInfo
	for _, dd := range deviceData {
		fields := make(map[string]any)
		for key, value := range dd.SensorData {
			fields[key] = value
		}
		deviceInfo, ok = deviceInfoMap[dd.DeviceID]
		if !ok {
			err = fmt.Errorf("could not find deviceInfo: %s", dd.DeviceID)
		}
		writeApi = t.influx.db.WriteAPI(testingInfluxOpts.Org, deviceInfo.Network)
		p := influxdb2.NewPoint(
			deviceInfo.Company,
			map[string]string{
				"deviceID": dd.DeviceID,
			},
			fields,
			dd.Timestamp,
		)
		writeApi.WritePoint(p)
	}
	if !ok {
		return err
	}
	writeApi.Flush()
	return nil
}

func deviceInfoMap(allDevicesInfo *deviceinfo.Schema) map[string]deviceinfo.DeviceInfo {
	deviceInfoMap := make(map[string]deviceinfo.DeviceInfo)
	var di deviceinfo.DeviceInfo
	for _, dd := range allDevicesInfo.DeviceNetworks {
		di = deviceInfoMap[dd.DeviceId]
		di.Network = dd.Network
		deviceInfoMap[dd.DeviceId] = di
	}
	for _, dd := range allDevicesInfo.DeviceCompanies {
		di = deviceInfoMap[dd.DeviceId]
		di.Company = dd.Company
		deviceInfoMap[dd.DeviceId] = di
	}
	return deviceInfoMap
}

func (t *TestingInflux) GetData(metadata deviceinfo.DeviceInfo) (
	DeviceDataSlice, error) {
	return t.influx.GetData(metadata)
}

func (t *TestingInflux) GetDataBoundary(deviceInfo deviceinfo.DeviceInfo) (
	DataBoundary, error) {
	return t.influx.GetDataBoundary(deviceInfo)
}
