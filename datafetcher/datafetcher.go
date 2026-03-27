package datafetcher

import (
	"time"

	deviceinfo "github.com/datafarm-software/datafarm-api/device-info"
)

type DeviceDataResponse struct {
	Body []DeviceData
}

type DeviceDataRequest struct {
	DeviceId    string   `path:"deviceId" pattern:"^[a-zA-Z0-9]{1,30}$" required:"true"`
	QueryFields []string `query:"queryField,explode" json:"queryFields" required:"true" doc:"If all QueryFields desired, set ?queryField=all. Multiple QueryFields supported using format: ?queryField=temperature&queryField=humidity."`
	Start       string   `query:"start" json:"start" required:"true"`
	Stop        string   `query:"stop" json:"stop" required:"false"`
}

type BatchDeviceDataRequest struct {
	DeviceId    string   `json:"deviceId" pattern:"^[a-zA-Z0-9]{1,30}$" required:"true"`
	QueryFields []string `json:"queryFields" required:"true"`
	Start       string   `json:"start" required:"true"`
	Stop        string   `json:"stop" required:"false"`
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
	Close() error
}
