package metadatafetcher

type Metadata struct {
	DeviceId, Company, Network string
	QueryRange                 any
	QueryFields                []string
}
