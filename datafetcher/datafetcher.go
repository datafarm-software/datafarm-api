package datafetcher

import (
	"time"

	deviceinfo "github.com/datafarm-software/datafarm-api/device-info"
)

type DeviceDataResponse struct {
	Status int
	Body   []DeviceData
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
	Results []DeviceData      `json:"results"`
	Errors  []DeviceDataError `json:"errors"`
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
	PrepareDb(*deviceinfo.Schema, []DeviceData) error
}

type DataFetcher interface {
	TestingDataFetcher
	GetData(metadata deviceinfo.DeviceInfo) ([]DeviceData, error)
	GetDataBoundary(metadata deviceinfo.DeviceInfo) (DataBoundary, error)
	Close() error
}
