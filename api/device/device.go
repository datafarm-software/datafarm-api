package device

type Schema struct {
	DeviceCompanies []DeviceToCompany
	DeviceNetworks  []DeviceToNetwork
	DeviceToQF      []DeviceToQueryFields
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

type Batch struct {
	DeviceIds []string `log:"deviceids" json:"deviceIds" pattern:"^[a-zA-Z0-9]{1,30}$" minItems:"2" maxItems:"5"`
}
