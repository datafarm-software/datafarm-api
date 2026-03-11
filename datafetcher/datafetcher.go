package datafetcher

import (
	"time"

	deviceinfo "github.com/geraud22/datafarm-api/device-info"
)

type DeviceDataRequest struct {
	QueryField string `json:"queryField" required:"true"`
	Start      string `json:"start" required:"true"`
	Stop       string `json:"stop" required:"false"`
}

type ConsolidatedDeviceData struct {
	DeviceData []DeviceData `json:"payload"`
}

type DeviceData struct {
	DeviceID   string    `json:"rtuid"`
	Timestamp  time.Time `json:"timestamp"`
	SensorData map[string]any
}

type TestingDataFetcher interface {
	PrepareDb(*deviceinfo.Schema, *ConsolidatedDeviceData) error
}

type DataFetcher interface {
	TestingDataFetcher
	GetData(metadata deviceinfo.DeviceInfo) (*ConsolidatedDeviceData, error)
	Close() error
}
