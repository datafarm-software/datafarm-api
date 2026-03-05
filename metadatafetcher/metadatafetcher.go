package metadatafetcher

import "github.com/geraud22/datafarm-api/authoriser"

type Metadata struct {
	DeviceId, Company, Network   string
	Start, Stop                  string
	QueryFields, AttachedSensors []string
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

type UserToken struct {
	Username, Token string
}

type Schema struct {
	DeviceCompanies []DeviceToCompany
	DeviceNetworks  []DeviceToNetwork
	DeviceToSensors []DeviceToSensor
	SensorToQF      []SensorToQueryFields
	UserTokens      []UserToken
}

type TestingMetadataFetcher interface {
	PrepareMetadataFetcher(Schema) error
	GetSnapshot() *Schema
}

type MetadataFetcher interface {
	TestingMetadataFetcher
	Close() error
	GetAttachedSensors(deviceId string) ([]string, error)
	GetQueryFields(attachedSensors []string) ([]string, error)
	GetCompany(deviceId string) (string, error)
	GetNetwork(deviceId string) (string, error)
	GetUser(token string) (string, error)
	StoreToken(authoriser.UserToken) error
	DeleteToken(authoriser.TokenResponse) error
}
