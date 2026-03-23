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

type QueryFields struct {
	DeviceId    string   `json:"deviceId"`
	QueryFields []string `json:"queryFields"`
}

type DeviceIds struct {
	Body []string
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
	// DeviceToSensors []DeviceToSensor
	DeviceToQF []DeviceToQueryFields
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
