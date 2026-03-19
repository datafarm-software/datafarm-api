package deviceinfo

type DeviceInfo struct {
	DeviceId, Company, Network   string
	Start, Stop                  string
	QueryFields, AttachedSensors []string
}

type QueryFields struct {
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

type DeviceToSensor struct {
	DeviceId        string
	AttachedSensors []string
}

type SensorToQueryFields struct {
	Sensor      string
	QueryFields []string
}

type Schema struct {
	DeviceCompanies []DeviceToCompany
	DeviceNetworks  []DeviceToNetwork
	DeviceToSensors []DeviceToSensor
	SensorToQF      []SensorToQueryFields
}

type TestingDeviceInfoFetcher interface {
	PrepareDeviceInfo(Schema) error
}

type DeviceInfoFetcher interface {
	TestingDeviceInfoFetcher
	Close() error
	GetAttachedSensors(deviceId string) ([]string, error)
	GetQueryFields(attachedSensors []string) ([]string, error)
	GetCompany(deviceId string) (string, error)
	GetNetwork(deviceId string) (string, error)
}
