package deviceinfo

var GeneralQueryFields = []string{
	"latitude", "longitude", "signal_strength", "rssi", "snr", "batv",
}

type DeviceInfo struct {
	DeviceId, Company, Network   string
	Start, Stop                  string
	QueryFields, AttachedSensors []string
}

type QueryFieldsResponse struct {
	Body QueryFields
}

type QueryFieldsRequest struct {
	DeviceId string `path:"deviceId" pattern:"^[a-zA-Z0-9]{1,30}$" required:"true"`
}

type BatchQueryFieldsRequest struct {
	Body []string `json:"deviceIds" doc:"deviceIds" pattern:"^[a-zA-Z0-9]{1,30}$"`
}

type QueryFieldsError struct {
	DeviceId string `json:"deviceId"`
	Error    string `json:"error"`
}

type BatchQueryFieldsResponse struct {
	Results []QueryFields      `json:"results"`
	Errors  []QueryFieldsError `json:"errors"`
}

type QueryFields struct {
	DeviceId    string   `json:"deviceId"`
	QueryFields []string `json:"queryFields"`
}

type DeviceIdsResponse struct {
	Body []string `json:"deviceIds" doc:"deviceIds"`
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
	GetCompany(deviceId string) (string, error)
	GetNetwork(deviceId string) (string, error)
}
