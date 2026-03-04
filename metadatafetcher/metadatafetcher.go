package metadatafetcher

type Metadata struct {
	DeviceId, Company, Network string
	QueryRange                 any
	QueryFields                []string
}

type MetadataFetcher interface {
	Close() error
	GetAttachedSensors(deviceId string) ([]string, error)
	GetQueryFields(attachedSensors []string) ([]string, error)
	GetCompany(deviceId string) (string, error)
	GetNetwork(deviceId string) (string, error)
	LinkTokenToUser(username, token string) error
}
