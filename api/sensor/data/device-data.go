package data

import (
	"encoding/csv"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/datafarm-software/datafarm-api/api/sensor"
)

var EmptySensorData = errors.New("empty sensor data")
var EmptyTimeZone = errors.New("empty time zone")

type SensorDataResponse struct {
	Status int
	Body   SensorDataSlice
}

type Timezone struct {
	Timezone string `log:"timezone" query:"timezone-return" json:"timezone-return" required:"false" pattern:"^(|[a-zA-Z]+/[a-zA-Z]+)$" doc:"Clients can specify a timezone for the returned SensorData. Supports IANA Timezone definitions eg. Africa/Johannesburg"`
}

func (t Timezone) IsEmpty() bool {
	return t.Timezone == ""
}

func (t Timezone) Location() (*time.Location, error) {
	if t.IsEmpty() {
		t.Timezone = "UTC"
	}
	return time.LoadLocation(string(t.Timezone))
}

type TimeFrame struct {
	Timezone
	Start string `log:"start" query:"start" json:"start" required:"true" doc:"Client specified timestamps are treated as inclusive in returned data. Client can specify a start time in two formats. 1. Relative Format eg. '-[0-9]{1,3}mo|m|h|d'. In Relative Format a stop time is not required. 2. RFC3339 Format. Now a Stop time is required. Start cannot be older than 90 days."`
	Stop  string `log:"stop" query:"stop" json:"stop" required:"false" doc:"Client specified timestamps are treated as inclusive in returned data. If Start is in RFC3339 Format, Stop field is required. Stop time must be later than Start. Stop can only ever be in RFC3339 Format."`
}

type SensorDataRequest struct {
	sensor.Hardware
	TimeFrame
}

type LatestSensorDataRequest struct {
	sensor.Hardware
	Timezone
}

type BatchLatestSensorDataRequest struct {
	Hardware []sensor.Hardware `json:"hardware" required:"true" minItems:"2" maxItems:"5"`
	Timezone
}

type BatchSensorDataRequest struct {
	Hardware []sensor.Hardware `json:"hardware" required:"true" minItems:"2" maxItems:"5"`
	TimeFrame
}

type BatchSensorDataResponse struct {
	Results SensorDataSlice     `json:"results"`
	Errors  []sensor.BatchError `json:"errors"`
}

func (b *BatchSensorDataResponse) Csv() (csvStr string, err error) {
	csvStr, _ = b.Results.Csv()
	errStr := strings.Builder{}
	for _, de := range b.Errors {
		fmt.Fprintf(&errStr, "%s,%s\n", de.DeviceId, de.Error)
	}
	csvStr += errStr.String()
	return
}

type Indexes []int

type CsvMarshaller interface {
	Csv() (string, error)
}

type CsvInfo struct {
	Headers         []string
	DeviceIdIndexes map[sensor.DeviceId]Indexes
	DeviceIds       sensor.DeviceIds
}

type SensorDataSlice []SensorData

func (d SensorDataSlice) CsvInfo() (csvInfo CsvInfo, err error) {
	csvInfo.Headers = make([]string, 0, len(d))
	csvInfo.DeviceIdIndexes = make(map[sensor.DeviceId]Indexes)
	if len(d) < 1 {
		return csvInfo, EmptySensorData
	}
	queryFieldSeen := make(map[string]bool)
	sorted := slices.Clone(d)
	slices.SortFunc(sorted, func(a, b SensorData) int {
		return a.Timestamp.Compare(b.Timestamp)
	})
	var id sensor.DeviceId
	idSeen := make(map[sensor.DeviceId]bool)
	for i, dd := range sorted {
		id = sensor.DeviceId(dd.DeviceID)
		csvInfo.DeviceIdIndexes[id] = append(
			csvInfo.DeviceIdIndexes[sensor.DeviceId(dd.DeviceID)], i)
		if !idSeen[id] {
			csvInfo.DeviceIds = append(csvInfo.DeviceIds, id)
			idSeen[id] = true
		}
		for qf := range dd.SensorData {
			if !queryFieldSeen[qf] {
				csvInfo.Headers = append(csvInfo.Headers, qf)
				queryFieldSeen[qf] = true
			}
		}
	}
	slices.Sort(csvInfo.Headers)
	slices.Sort(csvInfo.DeviceIds)
	return csvInfo, nil
}

func (d SensorDataSlice) Csv() (csvStr string, err error) {
	if len(d) < 1 {
		return csvStr, EmptySensorData
	}
	csvInfo, err := d.CsvInfo()
	if err != nil {
		return csvStr, fmt.Errorf("csvheaders: %v", err)
	}
	var str strings.Builder
	writer := csv.NewWriter(&str)
	blankStartingColumn := []string{""}
	blankStartingColumn = append(blankStartingColumn, csvInfo.Headers...)
	if err := writer.Write(blankStartingColumn); err != nil {
		return csvStr, fmt.Errorf("writing headers: %v", err)
	}
	var sensorData SensorData
	var indexes Indexes
OuterLoop:
	for _, deviceId := range csvInfo.DeviceIds {
		err = writeDeviceIdRow(string(deviceId), writer)
		if err != nil {
			err = fmt.Errorf("writing deviceid row: %v", err)
			break
		}
		indexes = csvInfo.DeviceIdIndexes[deviceId]
		for _, i := range indexes {
			if len(d) <= i {
				err = fmt.Errorf(
					"deviceid: %s, gave index out of range: len(%d) <= %d",
					deviceId, len(d), i)
				break OuterLoop
			}
			sensorData = d[i]
			err = writeDataRow(csvInfo.Headers, sensorData, writer)
			if err != nil {
				err = fmt.Errorf("writing row: %v", err)
				break OuterLoop
			}
		}
	}
	writer.Flush()
	return str.String(), err
}

func writeDeviceIdRow(deviceId string, writer *csv.Writer) (err error) {
	deviceIdRow := []string{string(deviceId)}
	return writer.Write(deviceIdRow)
}

func writeDataRow(queryFieldColumns []string, sensorData SensorData, writer *csv.Writer) error {
	row := []string{sensorData.Timestamp.Format(time.RFC3339)}
	var v float64
	var ok bool
	for _, qf := range queryFieldColumns {
		v, ok = sensorData.SensorData[qf]
		if ok {
			row = append(row, fmt.Sprintf("%.3f", v))
		} else {
			row = append(row, "")
		}
	}
	return writer.Write(row)
}

type SensorData struct {
	DeviceID   sensor.DeviceId    `json:"deviceId"`
	Timestamp  time.Time          `json:"timestamp" doc:"Timestamp will be in RFC3339 Format. Default timezone is UTC."`
	SensorData map[string]float64 `json:"sensorData"`
}

type DataBoundarySlice []DataBoundary

type DataBoundary struct {
	DeviceId sensor.DeviceId `json:"deviceId"`
	Start    time.Time       `json:"start"`
	Stop     time.Time       `json:"stop"`
}

type DataBoundaryRequest struct {
	sensor.DeviceIdParam
	Timezone
}
type DataBoundaryResponse struct {
	Status int
	Body   DataBoundary
}

type BatchDataBoundaryRequest struct {
	sensor.Batch
	Timezone
}

type BatchDataBoundaryResponse struct {
	Results []DataBoundary      `json:"results"`
	Errors  []sensor.BatchError `json:"errors"`
}

type DeviceLocationResponseSlice []DeviceLocationResponse

type DeviceLocationResponse struct {
	DeviceId  sensor.DeviceId `json:"deviceId"`
	Time      time.Time       `json:"time" doc:"Time the latest Location was reported."`
	Latitude  float64         `log:"latitude" json:"latitude"`
	Longitude float64         `log:"longitude" json:"longitude"`
}

type BatchLocationRequest struct {
	sensor.Batch
}

type BatchLocationResponse struct {
	Results []DeviceLocationResponse `json:"results"`
	Errors  []sensor.BatchError      `json:"errors"`
}

type TestingDataFetcher interface {
	PrepareDb(*sensor.Schema, SensorDataSlice) error
}

type Fetcher interface {
	TestingDataFetcher
	//NOTE: could return NoData, NoConnection
	GetData(metadata sensor.Device) (SensorDataSlice, error)
	//NOTE: could return NoData, NoConnection
	GetLatestData(metadata sensor.Device) (SensorData, error)
	//NOTE: could return NoData, NoConnection
	GetDataBoundary(metadata sensor.Device) (DataBoundary, error)
	//NOTE: could return NoLocation, NoConnection
	GetLocation(metadata sensor.Device) (DeviceLocationResponse, error)
	Close() error
}
