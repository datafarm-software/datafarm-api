package metadatafetcher

import "github.com/geraud22/datafarm-api/authoriser"

type Metadata struct {
	DeviceId, Company, Network   string
	Start, Stop                  string
	QueryFields, AttachedSensors []string
}

type Schema struct {
}

type TestingMetadataFetcher interface {
	PrepareMetadataFetcher(Schema) error
}

type MetadataFetcher interface {
	TestingMetadataFetcher
	Close() error
	GetAttachedSensors(deviceId string) ([]string, error)
	GetQueryFields(attachedSensors []string) ([]string, error)
	GetCompany(deviceId string) (string, error)
	GetNetwork(deviceId string) (string, error)
	StoreToken(authoriser.UserToken) error
	DeleteToken(authoriser.TokenResponse) error
}
