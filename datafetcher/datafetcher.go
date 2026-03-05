package datafetcher

import (
	"time"

	deviceinfo "github.com/geraud22/datafarm-api/device-info"
)

type DeviceDataRequest struct {
	DeviceId   string `path:"deviceId" pattern:"^[a-zA-Z0-9]{1,30}$" required:"true"`
	QueryField string `query:"queryField" required:"true"`
	Start      string `query:"start" required:"true"`
	Stop       string `query:"stop" required:"false"`
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
	PrepareDb(measurement string, cdd *ConsolidatedDeviceData) error
}

type DataFetcher interface {
	TestingDataFetcher
	GetData(metadata deviceinfo.DeviceInfo) (*ConsolidatedDeviceData, error)
	Close() error
}
