package info

import (
	"errors"

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

var NotFound = errors.New("not found")

type ScopeRestriction struct {
	Scope   Scope
	Company string
	Network string
}

type QueryFieldsResponse struct {
	Body QueryFields
}

type QueryFieldsRequest struct {
	DeviceId string `log:"deviceid" path:"deviceId" pattern:"^[a-zA-Z0-9]{1,30}$" required:"true"`
}

type BatchQueryFieldsRequest struct{ Body sensor.Batch }

type QueryFieldsError struct {
	DeviceId string `json:"deviceId"`
	Error    string `json:"error"`
}

type QueryFields struct {
	DeviceId    string   `json:"deviceId"`
	QueryFields []string `json:"queryFields"`
}

type BatchQueryFieldsResponse struct {
	Results []QueryFields      `json:"results"`
	Errors  []QueryFieldsError `json:"errors"`
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
	GetQueryFields(deviceId string) (QueryFields, error)
	//NOTE: could return err: NotFound
	GetCompany(deviceId string) (string, error)
	//NOTE: could return err: NotFound
	GetNetwork(deviceId string) (string, error)
	GetDevices(ScopeRestriction) ([]string, error)
}

type BadConnFetcher struct{}

func (b *BadConnFetcher) Close() error { return nil }
func (b *BadConnFetcher) GetQueryFields(deviceId string) (QueryFields, error) {
	return QueryFields{}, sensor.NoConnection
}

func (b *BadConnFetcher) GetCompany(deviceId string) (string, error) {
	return "", sensor.NoConnection
}

func (b *BadConnFetcher) GetNetwork(deviceId string) (string, error) {
	return "", sensor.NoConnection
}

func (b *BadConnFetcher) GetDevices(ScopeRestriction) ([]string, error) {
	return nil, sensor.NoConnection
}
