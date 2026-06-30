package datafetcher

import (
	"errors"
	"fmt"
	"time"

	deviceinfo "github.com/datafarm-software/datafarm-api/device-info"
)

var EmptyDeviceData = errors.New("empty device data")

type DeviceDataResponse struct {
	Status int
	Body   DeviceDataSlice
}

type DeviceDataRequest struct {
	DeviceId    string   `path:"deviceId" pattern:"^[a-zA-Z0-9]{1,30}$" required:"true"`
	QueryFields []string `query:"queryField,explode" json:"queryFields" required:"true" doc:"If all QueryFields desired, set ?queryField=all. Multiple QueryFields supported using format: ?queryField=temperature&queryField=humidity."`
	Start       string   `query:"start" json:"start" required:"true" doc:"User specified timestamps are treated as inclusive in returned data. Client can specify a start time in two formats. 1. Relative Format eg. '-[0-9]{1,3}mo|m|h|d'. In Relative Format a stop time is not required. 2. RFC3339 Format. Now a Stop time is required. Start cannot be older than 90 days."`
	Stop        string   `query:"stop" json:"stop" required:"false" doc:"User specified timestamps are treated as inclusive in returned data. If Start is in RFC3339 Format, Stop field is required. Stop can only ever be in RFC3339 Format."`
}

type BatchDeviceDataRequest struct {
	DeviceId    string   `json:"deviceId" pattern:"^[a-zA-Z0-9]{1,30}$" required:"true"`
	QueryFields []string `json:"queryFields" required:"true"`
	Start       string   `query:"start" json:"start" required:"true" doc:"User specified timestamps are treated as inclusive in returned data. Client can specify a start time in two formats. 1. Relative Format eg. '-[0-9]{1,3}mo|m|h|d'. In Relative Format a stop time is not required. 2. RFC3339 Format. Now a Stop time is required. Start cannot be older than 90 days."`
	Stop        string   `query:"stop" json:"stop" required:"false" doc:"User specified timestamps are treated as inclusive in returned data. If Start is in Relative Format, Stop field is not required. Stop can only ever be in RFC3339 Format."`
}

type DeviceDataError struct {
	DeviceId string `json:"deviceId"`
	Error    string `json:"error"`
}

type BatchDeviceDataResponse struct {
	Results DeviceDataSlice   `json:"results"`
	Errors  []DeviceDataError `json:"errors"`
}

type DeviceId string
type Indexes []int

type CsvInfo struct {
	Headers         []string
	DeviceIdIndexes map[DeviceId]Indexes
}

type DeviceDataSlice []DeviceData

func (d DeviceDataSlice) CsvInfo() (csvInfo CsvInfo, err error) {
	csvInfo.Headers = make([]string, 0, len(d))
	csvInfo.DeviceIdIndexes = make(map[DeviceId]Indexes)
	if len(d) < 1 {
		return csvInfo, EmptyDeviceData
	}
	queryFieldSeen := make(map[string]bool)
	for i, dd := range d {
		for qf, _ := range dd.SensorData {
			csvInfo.DeviceIdIndexes[DeviceId(dd.DeviceID)] = append(
				csvInfo.DeviceIdIndexes[DeviceId(dd.DeviceID)], i)
			if !queryFieldSeen[qf] {
				csvInfo.Headers = append(csvInfo.Headers, qf)
				queryFieldSeen[qf] = true
			}
		}
	}
	return csvInfo, nil
}

func (d DeviceDataSlice) Csv() (csvStr []string, err error) {
	// 	if len(d) < 1 {
	// 		return csvStr, EmptyDeviceData
	// 	}
	// 	headers, err := d.CsvInfo()
	// 	if err != nil {
	// 		return csvStr, fmt.Errorf("csvheaders: %v", err)
	// 	}
	// 	var str strings.Builder
	// 	writer := csv.NewWriter(&str)
	// 	if err := writer.Write(headers); err != nil {
	// 		return csvStr, fmt.Errorf("headers: %v", err)
	// 	}
	// for _, inv := range s.Invites {
	// 	if err := writer.Write(inv.ToRow()); err != nil {
	// 		err = fmt.Errorf("writing invite row: %v", err)
	// 		break
	// 	}
	// }
	// writer.Flush()
	// return str.String(), err
	return csvStr, fmt.Errorf("not implemented")
}

type DeviceData struct {
	DeviceID   string             `json:"deviceId"`
	Timestamp  time.Time          `json:"timestamp"`
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
