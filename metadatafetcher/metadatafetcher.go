package metadatafetcher

import "github.com/geraud22/datafarm-api/authoriser"

type Metadata struct {
	DeviceId, Company, Network string
	QueryRange                 any
	QueryFields                []string
}

type TestingMetadataFetcher interface {
	PrepareMetadataFetcher(Metadata) error
}

type MetadataFetcher interface {
	TestingMetadataFetcher
	Close() error
	GetAttachedSensors(deviceId string) ([]string, error)
	GetQueryFields(attachedSensors []string) ([]string, error)
	GetCompany(deviceId string) (string, error)
	GetNetwork(deviceId string) (string, error)
	StoreToken(authoriser.UserToken) error
}
