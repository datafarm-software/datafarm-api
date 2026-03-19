package datafetcher

import (
	"time"

	deviceinfo "github.com/datafarm-software/datafarm-api/device-info"
)

type DeviceDataRequest struct {
	QueryFields []string `json:"queryFields" required:"true"`
	Start       string   `json:"start" required:"true"`
	Stop        string   `json:"stop" required:"false"`
}

type DeviceData struct {
	DeviceID   string         `json:"deviceId"`
	Timestamp  time.Time      `json:"timestamp"`
	SensorData map[string]any `json:"sensorData"`
}

type TestingDataFetcher interface {
	PrepareDb(*deviceinfo.Schema, []DeviceData) error
}

type DataFetcher interface {
	TestingDataFetcher
	GetData(metadata deviceinfo.DeviceInfo) ([]DeviceData, error)
	Close() error
}
