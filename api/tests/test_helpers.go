package tests

import (
	"context"
	"fmt"
	"net/url"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humamux"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/datafarm-software/datafarm-api/api"
	"github.com/datafarm-software/datafarm-api/api/authstore"
	localhuma "github.com/datafarm-software/datafarm-api/api/huma"
	"github.com/datafarm-software/datafarm-api/api/redis"
	"github.com/datafarm-software/datafarm-api/api/sensor"
	"github.com/datafarm-software/datafarm-api/api/sensor/data"
	"github.com/datafarm-software/datafarm-api/api/tokenprovider"
	"github.com/datafarm-software/telemetry"
	"github.com/datafarm-software/telemetry/logging"
	"github.com/datafarm-software/telemetry/metering"
	"github.com/datafarm-software/telemetry/middlewares"
	"github.com/datafarm-software/telemetry/tracing"
	"github.com/google/go-cmp/cmp"
	"github.com/gorilla/mux"
	"github.com/mitchellh/reflectwalk"
	"github.com/stretchr/testify/require"
)

const (
	RegisteredUsername                  = "user1"
	UnregisteredUsername                = "user2"
	RegisteredPassword                  = "@Password1"
	UnregisteredPassword                = "@Password2"
	RegisteredCompany                   = "company"
	AnotherRegisteredCompany            = "company2"
	OtherCompanyThanDevice              = "othercompany"
	RegisteredNetwork                   = "RegisteredNetwork"
	AnotherRegisteredNetwork            = "RegisteredNetwork2"
	RegisteredDeviceId                  = "device1"
	AnotherRegisteredDeviceId           = "device2"
	UnregisteredDeviceId                = "unregistered1"
	InvalidDeviceId                     = "!+)$"
	RegisteredQueryField                = "temperature"
	AnotherRegisteredQueryField         = "humidity"
	RegisteredSensor                    = "weather-sensor"
	ValidToken                          = "someToken0"
	InvalidToken                        = "invalidToken0"
	RelativeStart                       = "-6h"
	RelativeMoreThanNinetyDays          = "-91d"
	ValidTimezone                       = "Africa/Johannesburg"
	InvalidTimezone                     = "$ome/Wr0ng/Timezone"
	Latitude                    float64 = -25.496973
	Longitude                   float64 = 31.558536
	AnotherLatitude             float64 = -26.496973
	AnotherLongitude            float64 = 32.558536
)

var (
	MoreThanNinetyDays       = time.Now().UTC().Add(-91 * 24 * time.Hour).Format(time.RFC3339)
	Start                    = time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339)
	StartGreaterThanStop     = time.Now().UTC().Add(1 * time.Hour).Format(time.RFC3339)
	FutureStart              = time.Now().UTC().Add(1 * time.Hour).Format(time.RFC3339)
	Stop                     = time.Now().UTC().Format(time.RFC3339)
	StopInFuture             = time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339)
	OutsideTimeRange         = time.Now().UTC().Add(-25 * time.Hour)
	InsideTimeRange          = time.Now().UTC().Add(-2 * time.Hour)
	AlsoInsideTimeRange      = time.Now().UTC().Add(-1 * time.Hour)
	RegisteredCompanyDevices = []string{RegisteredDeviceId}
)

var considerTimeZone = cmp.Comparer(func(x, y time.Time) bool {
	return x.Equal(y) &&
		x.Location().String() == y.Location().String()
})
var cmpOpts = []cmp.Option{considerTimeZone}

type MockApi struct {
	mockDeviceInfo  sensor.Schema
	mockAuthStore   authstore.Schema
	mockDataFetcher []data.SensorData
	mockTokens      map[string]bool
}

func setupHuma(t *testing.T, api *api.Api) humatest.TestAPI {
	config := localhuma.Config(localhuma.Production)
	router := mux.NewRouter()
	humaApi := humamux.New(router, config)
	mwProvider, _ := middlewares.NewHumaProvider(api.Telemetry)
	mws := mwProvider.TelemetryMiddlewares()
	mws = append(mws, api.VerifyToken)
	mws = slices.Insert(mws, 0, api.RateLimit)
	localhuma.SetupApiOperations(humaApi, api, mws)
	return humatest.Wrap(t, humaApi)
}

func (m MockApi) Setup(t *testing.T) (*api.Api, CloseFunc) {
	a := &api.Api{}
	a.TokenProvider = &tokenprovider.MockTokenProvider{
		Tokens:    m.mockTokens,
		Increment: len(m.mockTokens),
	}
	a.Telemetry = &telemetry.Telemetry{
		Logger: logging.MockLogger(),
		Meter:  metering.MockMeter(),
		Tracer: tracing.MockTracer(),
	}
	db, err := miniredis.Run()
	require.Nil(t, err)
	testingRedis, err := redis.NewTestingRedis(db.Addr())
	require.Nil(t, err)
	a.DeviceInfo = testingRedis
	a.AuthStore = testingRedis
	err = testingRedis.PrepareDeviceInfo(m.mockDeviceInfo)
	require.Nil(t, err)
	err = testingRedis.PrepareAuthStore(m.mockAuthStore)
	require.Nil(t, err)
	if m.mockDataFetcher != nil {
		a.DataFetcher, err = data.NewTestingInflux("../../config.yml")
		require.Nil(t, err)
		err = a.DataFetcher.PrepareDb(&m.mockDeviceInfo, m.mockDataFetcher)
		require.Nil(t, err)
	}
	return a, func() {
		db.Close()
		if a.TokenProvider != nil {
			err = a.TokenProvider.Close()
			if err != nil {
				t.Logf("tokenprovider close: %v", err)
			}
		}
		err = testingRedis.Close()
		if err != nil {
			t.Logf("testingRedis close: %v", err)
		}
		if a.DataFetcher != nil {
			err = a.DataFetcher.Close()
			if err != nil {
				t.Logf("datafetcher close: %v", err)
			}
		}
		if a.Telemetry != nil {
			err = a.Telemetry.Close(context.Background())
			if err != nil {
				t.Logf("telemetry close: %v", err)
			}
		}
	}
}

type GetSensorDataTest struct {
	token    string
	deviceId string
	*data.LatestSensorDataRequest
	*data.SensorDataRequest
	*data.BatchSensorDataRequest
	wantStatus int
	wantErr    bool
}

type CloseFunc func()

type queryWalker struct {
	strings.Builder
}

func (w *queryWalker) Struct(reflect.Value) error { return nil }

func (w *queryWalker) StructField(
	field reflect.StructField,
	value reflect.Value,
) error {
	tag := field.Tag.Get("query")
	if tag == "" {
		return nil
	}
	tag = strings.ReplaceAll(tag, ",explode", "")
	value = reflect.Indirect(value)
	switch value.Kind() {
	case reflect.String:
		if value.String() != "" {
			fmt.Fprintf(&w.Builder, `%s=%s&`, tag, url.QueryEscape(value.String()))
		}
	case reflect.Slice:
		for i := range value.Len() {
			elem := value.Index(i)
			if elem.Kind() != reflect.String || elem.String() == "" {
				continue
			}
			fmt.Fprintf(&w.Builder, `%s=%s&`, tag, url.QueryEscape(elem.String()))
		}
	}
	return nil
}

func makeQueryParams[T any](dr T, t *testing.T) string {
	w := &queryWalker{strings.Builder{}}
	fmt.Fprintf(&w.Builder, "?")
	err := reflectwalk.Walk(dr, w)
	require.Nil(t, err)
	urlQuery := w.String()
	lastAnd := strings.LastIndex(urlQuery, "&")
	urlQuery = urlQuery[:lastAnd]
	return urlQuery
}
