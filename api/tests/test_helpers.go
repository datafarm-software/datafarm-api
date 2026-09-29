package tests

import (
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humamux"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/datafarm-software/datafarm-api/api"
	"github.com/datafarm-software/datafarm-api/api/authstore"
	"github.com/datafarm-software/datafarm-api/api/datafetcher"
	deviceinfo "github.com/datafarm-software/datafarm-api/api/device-info"
	localhuma "github.com/datafarm-software/datafarm-api/api/huma"
	"github.com/datafarm-software/datafarm-api/api/redis"
	"github.com/datafarm-software/datafarm-api/api/tokenprovider"
	"github.com/datafarm-software/telemetry/logging"
	"github.com/datafarm-software/telemetry/metering"
	"github.com/datafarm-software/telemetry/tracing"
	"github.com/google/go-cmp/cmp"
	"github.com/gorilla/mux"
	"github.com/mitchellh/reflectwalk"
	"github.com/stretchr/testify/require"
)

const RegisteredUsername = "user1"
const UnregisteredUsername = "user2"
const RegisteredPassword = "@Password1"
const UnregisteredPassword = "@Password2"
const RegisteredCompany = "company"
const AnotherRegisteredCompany = "company2"
const OtherCompanyThanDevice = "othercompany"
const RegisteredNetwork = "RegisteredNetwork"
const AnotherRegisteredNetwork = "RegisteredNetwork2"
const RegisteredDeviceId = "device1"
const AnotherRegisteredDeviceId = "device2"
const UnregisteredDeviceId = "unregistered1"
const InvalidDeviceId = "!+)$"
const RegisteredQueryField = "temperature"
const AnotherRegisteredQueryField = "humidity"
const RegisteredSensor = "weather-sensor"
const ValidToken = "someToken0"
const InvalidToken = "invalidToken0"
const RelativeStart = "-6h"
const RelativeMoreThanNinetyDays = "-91d"
const ValidTimezone = "Africa/Johannesburg"
const InvalidTimezone = "$ome/Wr0ng/Timezone"
const Latitude float64 = -25.496973
const Longitude float64 = 31.558536
const AnotherLatitude float64 = -26.496973
const AnotherLongitude float64 = 32.558536

var MoreThanNinetyDays = time.Now().UTC().Add(-91 * 24 * time.Hour).Format(time.RFC3339)
var Start = time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339)
var StartGreaterThanStop = time.Now().UTC().Add(1 * time.Hour).Format(time.RFC3339)
var FutureStart = time.Now().UTC().Add(1 * time.Hour).Format(time.RFC3339)
var Stop = time.Now().UTC().Format(time.RFC3339)
var StopInFuture = time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339)
var OutsideTimeRange = time.Now().UTC().Add(-25 * time.Hour)
var InsideTimeRange = time.Now().UTC().Add(-2 * time.Hour)
var AlsoInsideTimeRange = time.Now().UTC().Add(-1 * time.Hour)
var RegisteredCompanyDevices = []string{RegisteredDeviceId}

var considerTimeZone = cmp.Comparer(func(x, y time.Time) bool {
	return x.Equal(y) &&
		x.Location().String() == y.Location().String()
})
var cmpOpts = []cmp.Option{considerTimeZone}

type MockApi struct {
	mockDeviceInfo  deviceinfo.Schema
	mockAuthStore   authstore.Schema
	mockDataFetcher []datafetcher.SensorData
	mockTokens      map[string]bool
}

func setupHuma(t *testing.T, api *api.Api) humatest.TestAPI {
	config := localhuma.Config(localhuma.Production)
	router := mux.NewRouter()
	humaApi := humamux.New(router, config)
	localhuma.SetupApiOperations(humaApi, api)
	return humatest.Wrap(t, humaApi)
}

func (m MockApi) Setup(t *testing.T) (*api.Api, CloseFunc) {
	a := &api.Api{}
	a.TokenProvider = &tokenprovider.MockTokenProvider{
		Tokens:    m.mockTokens,
		Increment: len(m.mockTokens),
	}
	a.Logger = logging.MockLogger()
	a.Meter = metering.MockMeter()
	a.Tracer = tracing.MockTracer()
	db, err := miniredis.Run()
	require.Nil(t, err)
	testingRedis, err := redis.NewTestingRedis(db.Addr())
	require.Nil(t, err)
	a.DeviceInfo = testingRedis
	a.AuthStore = testingRedis
	a.DataFetcher, err = datafetcher.NewTestingInflux("../../config.yml")
	require.Nil(t, err)
	err = a.DataFetcher.PrepareDb(&m.mockDeviceInfo, m.mockDataFetcher)
	require.Nil(t, err)
	err = testingRedis.PrepareDeviceInfo(m.mockDeviceInfo)
	require.Nil(t, err)
	err = testingRedis.PrepareAuthStore(m.mockAuthStore)
	require.Nil(t, err)
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
	}
}

type GetSensorDataTest struct {
	token    string
	deviceId string
	*datafetcher.LatestSensorDataRequest
	*datafetcher.SensorDataRequest
	*datafetcher.BatchSensorDataRequest
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
