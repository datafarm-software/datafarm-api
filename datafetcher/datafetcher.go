package datafetcher

import (
	"time"

	"github.com/geraud22/datafarm-api/metadatafetcher"
)

type ConsolidatedDeviceData struct {
	DeviceData []DeviceData `json:"payload"`
}

type DeviceData struct {
	DeviceID   string    `json:"rtuid"`
	Timestamp  time.Time `json:"timestamp"`
	SensorData map[string]any
}

type DataFetcher interface {
	GetData(metadata metadatafetcher.Metadata) (*ConsolidatedDeviceData, error)
	FormatQueryRange(startTime, stopTime string) (any, error)
	Close() error
}
