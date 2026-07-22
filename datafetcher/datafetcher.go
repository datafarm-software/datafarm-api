package datafetcher

import (
	"encoding/csv"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	deviceinfo "github.com/datafarm-software/datafarm-api/device-info"
)

var EmptyDeviceData = errors.New("empty device data")

type DeviceDataResponse struct {
	Status int
	Body   DeviceDataSlice
}

type Hardware struct {
	DeviceId    string   `json:"deviceId" path:"deviceId" pattern:"^[a-zA-Z0-9]{1,30}$" required:"true"`
	QueryFields []string `query:"queryField,explode" json:"queryFields" required:"true" doc:"If all QueryFields desired, set ?queryField=all. Multiple QueryFields supported using format: ?queryField=temperature&queryField=humidity."`
}

type TimeFrame struct {
	Start    string `query:"start" json:"start" required:"true" doc:"Client specified timestamps are treated as inclusive in returned data. Client can specify a start time in two formats. 1. Relative Format eg. '-[0-9]{1,3}mo|m|h|d'. In Relative Format a stop time is not required. 2. RFC3339 Format. Now a Stop time is required. Start cannot be older than 90 days."`
	Stop     string `query:"stop" json:"stop" required:"false" doc:"Client specified timestamps are treated as inclusive in returned data. If Start is in RFC3339 Format, Stop field is required. Stop can only ever be in RFC3339 Format."`
	Timezone string `query:"timezone-return" json:"timezone-return" required:"false" pattern:"^(|[a-zA-Z]+/[a-zA-Z]+)$" doc:"Clients can specify a timezone for the returned SensorData. Supports IANA Timezone definitions eg. Africa/Johannesburg"`
}

type DeviceDataRequest struct {
	Hardware
	TimeFrame
}

type BatchDeviceDataRequest struct {
	Hardware []Hardware `json:"hardware" required:"true"`
	TimeFrame
}

type DeviceDataError struct {
	DeviceId string `json:"deviceId"`
	Error    string `json:"error"`
}

type BatchDeviceDataResponse struct {
	Results DeviceDataSlice   `json:"results"`
	Errors  []DeviceDataError `json:"errors"`
}

func (b *BatchDeviceDataResponse) Csv() (csvStr string, err error) {
	csvStr, _ = b.Results.Csv()
	errStr := strings.Builder{}
	for _, de := range b.Errors {
		fmt.Fprintf(&errStr, "%s,%s\n", de.DeviceId, de.Error)
	}
	csvStr += errStr.String()
	return
}

type DeviceId string
type Indexes []int

type CsvMarshaller interface {
	Csv() (string, error)
}

type CsvInfo struct {
	Headers         []string
	DeviceIdIndexes map[DeviceId]Indexes
	DeviceIds       []DeviceId
}

type DeviceDataSlice []DeviceData

func (d DeviceDataSlice) CsvInfo() (csvInfo CsvInfo, err error) {
	csvInfo.Headers = make([]string, 0, len(d))
	csvInfo.DeviceIdIndexes = make(map[DeviceId]Indexes)
	if len(d) < 1 {
		return csvInfo, EmptyDeviceData
	}
	queryFieldSeen := make(map[string]bool)
	sorted := slices.Clone(d)
	slices.SortFunc(sorted, func(a, b DeviceData) int {
		return a.Timestamp.Compare(b.Timestamp)
	})
	var id DeviceId
	idSeen := make(map[DeviceId]bool)
	for i, dd := range sorted {
		id = DeviceId(dd.DeviceID)
		csvInfo.DeviceIdIndexes[id] = append(
			csvInfo.DeviceIdIndexes[DeviceId(dd.DeviceID)], i)
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

func (d DeviceDataSlice) Csv() (csvStr string, err error) {
	if len(d) < 1 {
		return csvStr, EmptyDeviceData
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
	var deviceData DeviceData
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
			deviceData = d[i]
			err = writeDataRow(csvInfo.Headers, deviceData, writer)
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

func writeDataRow(queryFieldColumns []string, deviceData DeviceData, writer *csv.Writer) error {
	row := []string{deviceData.Timestamp.Format(time.RFC3339)}
	var v float64
	var ok bool
	for _, qf := range queryFieldColumns {
		v, ok = deviceData.SensorData[qf]
		if ok {
			row = append(row, fmt.Sprintf("%.3f", v))
		} else {
			row = append(row, "")
		}
	}
	return writer.Write(row)
}

type DeviceData struct {
	DeviceID   string             `json:"deviceId"`
	Timestamp  time.Time          `json:"timestamp" doc:"Timestamp will be in UTC timezone and RFC3339 Format."`
	SensorData map[string]float64 `json:"sensorData"`
}

type DataBoundary struct {
	DeviceId string    `json:"deviceId"`
	Start    time.Time `json:"start"`
	Stop     time.Time `json:"stop"`
}
type DataBoundaryRequest struct {
	DeviceId string `path:"deviceId" pattern:"^[a-zA-Z0-9]{1,30}$" required:"true"`
}
type DataBoundaryResponse struct{ Body DataBoundary }

type TestingDataFetcher interface {
	PrepareDb(*deviceinfo.Schema, DeviceDataSlice) error
}

type DataFetcher interface {
	TestingDataFetcher
	GetData(metadata deviceinfo.DeviceInfo) (DeviceDataSlice, error)
	GetDataBoundary(metadata deviceinfo.DeviceInfo) (DataBoundary, error)
	Close() error
}
