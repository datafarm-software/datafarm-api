package sensor

import (
	"context"
	"errors"
	"time"
)

var NoData = errors.New("No Data")
var NoConnection = errors.New("No Connection")
var NoLocation = errors.New("No Location")

type Device struct {
	QueryFields                []string
	Timezone                   *time.Location
	DeviceId, Company, Network string
	Start, Stop                string
}

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

// NOTE: this is exactly the same as deviceinfo.QueryFieldsError struct
type BatchError struct {
	DeviceId string `json:"deviceId"`
	Error    string `json:"error"`
}

type BatchItem interface {
	DeviceId() string
}

type BatchResult[T any] struct {
	Results               []T
	Errors                []BatchError
	OnlyDataMissingErrors bool
}

func BatchFactory[Item BatchItem, Request, Result any](
	ctx context.Context,
	items []Item, makeRequest func(Item) Request,
	get func(context.Context, Request) ([]Result, error),
) (batch BatchResult[Result], err error) {
	batch = BatchResult[Result]{
		Results:               make([]Result, 0, len(items)),
		Errors:                make([]BatchError, 0, len(items)),
		OnlyDataMissingErrors: true,
	}
	var req Request
	var results []Result
	for _, it := range items {
		req = makeRequest(it)
		results, err = get(ctx, req)
		if err == nil {
			batch.Results = append(batch.Results, results...)
			continue
		}
		if errors.Is(err, NoConnection) {
			return BatchResult[Result]{}, err
		}
		if !errors.Is(err, NoData) {
			batch.OnlyDataMissingErrors = false
		}
		batch.Errors = append(batch.Errors, BatchError{
			DeviceId: it.DeviceId(),
			Error:    err.Error(),
		})
	}
	return batch, nil
}
