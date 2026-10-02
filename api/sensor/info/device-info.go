package info

import (
	"github.com/datafarm-software/datafarm-api/api/sensor"
)

var GeneralQueryFields = []string{
	"latitude", "longitude", "signal_strength", "rssi", "snr", "batv",
}

type Scope int

const (
	DevicesInCompanyInNetwork Scope = iota
	DevicesInNetwork
	AllDevices
)

type ScopeRestriction struct {
	Scope   Scope
	Company string
	Network string
}

type QueryFieldsResponse struct {
	Body QueryFields
}

type BatchQueryFieldsRequest struct{ Body sensor.Batch }

type QueryFieldSlice []QueryFields

type QueryFields struct {
	DeviceId    sensor.DeviceId `json:"deviceId"`
	QueryFields []string        `json:"queryFields"`
}

type BatchQueryFieldsResponse struct {
	Results []QueryFields       `json:"results"`
	Errors  []sensor.BatchError `json:"errors"`
}

type DeviceIdsResponse struct {
	DeviceIds []string `json:"deviceIds" doc:"deviceIds" pattern:"^[a-zA-Z0-9]{1,30}$"`
}

type TestingDeviceInfoFetcher interface {
	PrepareDeviceInfo(sensor.Schema) error
}

type Fetcher interface {
	TestingDeviceInfoFetcher
	Close() error
	//NOTE: could return err: NotFound, NoConnection
	GetQueryFields(sensor.DeviceId) (QueryFields, error)
	//NOTE: could return err: NotFound, NoConnection
	GetCompany(sensor.DeviceId) (string, error)
	//NOTE: could return err: NotFound, NoConnection
	GetNetwork(sensor.DeviceId) (string, error)
	//NOTE: could return err: NoConnection
	GetDevices(ScopeRestriction) ([]string, error)
}

type BadConnFetcher struct{}

func (b *BadConnFetcher) PrepareDeviceInfo(sensor.Schema) error { return nil }
func (b *BadConnFetcher) Close() error                          { return nil }
func (b *BadConnFetcher) GetQueryFields(deviceId sensor.DeviceId) (QueryFields, error) {
	return QueryFields{}, sensor.NoConnection
}

func (b *BadConnFetcher) GetCompany(deviceId sensor.DeviceId) (string, error) {
	return "", sensor.NoConnection
}

func (b *BadConnFetcher) GetNetwork(deviceId sensor.DeviceId) (string, error) {
	return "", sensor.NoConnection
}

func (b *BadConnFetcher) GetDevices(ScopeRestriction) ([]string, error) {
	return nil, sensor.NoConnection
}
