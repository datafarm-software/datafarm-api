package deviceinfo

import (
	"errors"
	"time"
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

type DeviceInfo struct {
	QueryFields                []string
	Timezone                   *time.Location
	DeviceId, Company, Network string
	Start, Stop                string
}

type QueryFieldsResponse struct {
	Body QueryFields
}

type QueryFieldsRequest struct {
	DeviceId string `path:"deviceId" pattern:"^[a-zA-Z0-9]{1,30}$" required:"true"`
}

type BatchQueryFieldsRequest struct{ Body DeviceBatch }

type DeviceBatch struct {
	DeviceIds []string `json:"deviceIds" pattern:"^[a-zA-Z0-9]{1,30}$" minItems:"2" maxItems:"5"`
}

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
	Body []string `json:"deviceIds" doc:"deviceIds" pattern:"^[a-zA-Z0-9]{1,30}$"`
}

type DeviceToCompany struct {
	DeviceId string
	Company  string
}

type DeviceToNetwork struct {
	DeviceId string
	Network  string
}

type DeviceToQueryFields struct {
	DeviceId    string
	QueryFields []string
}

type Schema struct {
	DeviceCompanies []DeviceToCompany
	DeviceNetworks  []DeviceToNetwork
	DeviceToQF      []DeviceToQueryFields
}

type TestingDeviceInfoFetcher interface {
	PrepareDeviceInfo(Schema) error
}

type DeviceInfoFetcher interface {
	TestingDeviceInfoFetcher
	Close() error
	GetQueryFields(deviceId string) (QueryFields, error)
	//NOTE: could return err: NotFound
	GetCompany(deviceId string) (string, error)
	//NOTE: could return err: NotFound
	GetNetwork(deviceId string) (string, error)
	GetDevices(ScopeRestriction) ([]string, error)
}
