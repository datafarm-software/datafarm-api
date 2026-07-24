package tests

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humamux"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/datafarm-software/datafarm-api/api"
	localhuma "github.com/datafarm-software/datafarm-api/api/huma"
	"github.com/datafarm-software/datafarm-api/authstore"
	"github.com/datafarm-software/datafarm-api/datafetcher"
	deviceinfo "github.com/datafarm-software/datafarm-api/device-info"
	"github.com/datafarm-software/datafarm-api/redis"
	"github.com/datafarm-software/datafarm-api/tokenprovider"
	"github.com/google/go-cmp/cmp"
	"github.com/gorilla/mux"
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
var a = &api.Api{}

var considerTimeZone = cmp.Comparer(func(x, y time.Time) bool {
	return x.Equal(y) &&
		x.Location().String() == y.Location().String()
})
var cmpOpts = []cmp.Option{considerTimeZone}

func TestLogin(t *testing.T) {
	tests := map[string]struct {
		wantErr            bool
		wantStatus         int
		wantToken          string
		username, password string
		mockAuthStore      authstore.Schema
		mockDeviceInfo     deviceinfo.Schema
	}{

		"successfully login": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			wantToken:  ValidToken,
			username:   RegisteredUsername,
			password:   RegisteredPassword,
			mockAuthStore: authstore.Schema{
				UserInfo: []authstore.UserInfo{
					{
						Username: RegisteredUsername,
						Company:  RegisteredCompany,
						Role:     int(authstore.User),
						Password: RegisteredPassword,
						Network:  RegisteredNetwork,
					},
				},
			},
		},

		"deny access": {
			wantErr:    true,
			wantStatus: http.StatusUnauthorized,
			username:   UnregisteredUsername,
			password:   UnregisteredPassword,
			mockAuthStore: authstore.Schema{
				UserInfo: []authstore.UserInfo{
					{
						Username: RegisteredUsername,
						Company:  RegisteredCompany,
						Role:     int(authstore.User),
						Password: RegisteredPassword,
						Network:  RegisteredNetwork,
					},
				},
			},
		},
	}

	db, err := miniredis.Run()
	require.Nil(t, err)
	defer db.Close()
	humaTest := setupHuma(t)
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			testingRedis, err := redis.NewTestingRedis(db.Addr())
			require.Nil(t, err)
			defer testingRedis.Close()
			a.DeviceInfo = testingRedis
			a.AuthStore = testingRedis
			err = testingRedis.PrepareDeviceInfo(tc.mockDeviceInfo)
			require.Nil(t, err)
			err = testingRedis.PrepareAuthStore(tc.mockAuthStore)
			require.Nil(t, err)
			a.TokenProvider = &tokenprovider.MockTokenProvider{}
			defer a.TokenProvider.Close()
			encodedDetails := base64.StdEncoding.EncodeToString(
				[]byte(tc.username + ":" + tc.password))
			resp := humaTest.Post("/login",
				fmt.Sprintf("Authorization: Basic %s", encodedDetails))
			if resp.Code != tc.wantStatus {
				t.Fatalf("wantStatus: %d, response status: %d", tc.wantStatus, resp.Code)
			}
			if !tc.wantErr {
				tokens := a.AuthStore.GetActiveTokens()
				if len(tokens) != 1 {
					t.Fatalf("expected a stored user token, got len: %d", len(tokens))
				}
				if tokens[0].Token != ValidToken {
					t.Fatalf("expected token: %s, got token: %v",
						ValidToken, tokens[0].Token)
				}
			}
		})
	}
}

func TestGetSensorData(t *testing.T) {
	tests := map[string]struct {
		want []datafetcher.SensorData
		gsdt GetSensorDataTest
	}{

		"successfully get deviceid data": {
			want: []datafetcher.SensorData{
				{
					DeviceID:  RegisteredDeviceId,
					Timestamp: InsideTimeRange,
					SensorData: map[string]float64{
						RegisteredQueryField: 23,
					},
				},
			},
			gsdt: GetSensorDataTest{
				wantErr:    false,
				wantStatus: http.StatusOK,
				mockAuthStore: authstore.Schema{
					UserInfo: []authstore.UserInfo{
						{
							Username: RegisteredUsername,
							Company:  RegisteredCompany,
							Role:     int(authstore.User),
							Password: RegisteredPassword,
							Network:  RegisteredNetwork,
						},
					},
					UserTokens: []authstore.UserToken{
						{Username: RegisteredUsername, Token: ValidToken},
					},
				},
				mockDataFetcher: []datafetcher.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField: 23,
						},
					},
				},
				mockDeviceInfo: deviceinfo.Schema{
					DeviceCompanies: []deviceinfo.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []deviceinfo.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []deviceinfo.DeviceToQueryFields{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
				token:    ValidToken,
				deviceId: RegisteredDeviceId,
				deviceRequest: datafetcher.SensorDataRequest{
					Hardware: datafetcher.Hardware{
						QueryFields: []string{RegisteredQueryField},
					},
					TimeFrame: datafetcher.TimeFrame{
						Start: RelativeStart,
					},
				},
			},
		},

		"successfully get deviceid data in Africa/Johannesburg timezone": {
			want: []datafetcher.SensorData{
				{
					DeviceID:  RegisteredDeviceId,
					Timestamp: InsideTimeRange.Local(),
					SensorData: map[string]float64{
						RegisteredQueryField: 23,
					},
				},
			},
			gsdt: GetSensorDataTest{
				wantErr:    false,
				wantStatus: http.StatusOK,
				mockAuthStore: authstore.Schema{
					UserInfo: []authstore.UserInfo{
						{
							Username: RegisteredUsername,
							Company:  RegisteredCompany,
							Role:     int(authstore.User),
							Password: RegisteredPassword,
							Network:  RegisteredNetwork,
						},
					},
					UserTokens: []authstore.UserToken{
						{Username: RegisteredUsername, Token: ValidToken},
					},
				},
				mockDataFetcher: []datafetcher.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField: 23,
						},
					},
				},
				mockDeviceInfo: deviceinfo.Schema{
					DeviceCompanies: []deviceinfo.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []deviceinfo.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []deviceinfo.DeviceToQueryFields{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
				token:    ValidToken,
				deviceId: RegisteredDeviceId,
				deviceRequest: datafetcher.SensorDataRequest{
					Hardware: datafetcher.Hardware{QueryFields: []string{RegisteredQueryField}},
					TimeFrame: datafetcher.TimeFrame{
						Start:    RelativeStart,
						Timezone: datafetcher.Timezone{Timezone: "Africa/Johannesburg"},
					},
				},
			},
		},

		"invalid timezone requested so unprocessable": {
			want: nil,
			gsdt: GetSensorDataTest{wantErr: true,
				wantStatus: http.StatusUnprocessableEntity,
				mockAuthStore: authstore.Schema{
					UserInfo: []authstore.UserInfo{
						{
							Username: RegisteredUsername,
							Company:  RegisteredCompany,
							Role:     int(authstore.User),
							Password: RegisteredPassword,
							Network:  RegisteredNetwork,
						},
					},
					UserTokens: []authstore.UserToken{
						{Username: RegisteredUsername, Token: ValidToken},
					},
				},
				mockDataFetcher: []datafetcher.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField: 23,
						},
					},
				},
				mockDeviceInfo: deviceinfo.Schema{
					DeviceCompanies: []deviceinfo.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []deviceinfo.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []deviceinfo.DeviceToQueryFields{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				token: ValidToken,
				mockTokens: map[string]bool{
					ValidToken: true,
				},
				deviceId: RegisteredDeviceId,
				deviceRequest: datafetcher.SensorDataRequest{
					Hardware: datafetcher.Hardware{QueryFields: []string{RegisteredQueryField}},
					TimeFrame: datafetcher.TimeFrame{
						Start:    RelativeStart,
						Timezone: datafetcher.Timezone{Timezone: "$ome/Wr0ng/Timezone"},
					},
				},
			},
		},

		"unprocessable because more than 20 queryFields requested": {
			want: nil,
			gsdt: GetSensorDataTest{
				wantErr:    true,
				wantStatus: http.StatusUnprocessableEntity,
				mockAuthStore: authstore.Schema{
					UserInfo: []authstore.UserInfo{
						{
							Username: RegisteredUsername,
							Company:  RegisteredCompany,
							Role:     int(authstore.User),
							Password: RegisteredPassword,
							Network:  RegisteredNetwork,
						},
					},
					UserTokens: []authstore.UserToken{
						{Username: RegisteredUsername, Token: ValidToken},
					},
				},
				mockDataFetcher: []datafetcher.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField:        23,
							AnotherRegisteredQueryField: 24,
							"queryField3":               25,
							"queryField4":               26,
							"queryField5":               27,
							"queryField6":               28,
							"queryField7":               29,
							"queryField8":               30,
							"queryField9":               31,
							"queryField10":              32,
							"queryField11":              33,
							"queryField12":              34,
							"queryField13":              35,
							"queryField14":              36,
							"queryField15":              37,
							"queryField16":              38,
							"queryField17":              39,
							"queryField18":              40,
							"queryField19":              41,
							"queryField20":              42,
							"queryField21":              43,
						},
					},
				},
				mockDeviceInfo: deviceinfo.Schema{
					DeviceCompanies: []deviceinfo.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []deviceinfo.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []deviceinfo.DeviceToQueryFields{
						{
							DeviceId: RegisteredDeviceId,
							QueryFields: []string{
								RegisteredQueryField,
								AnotherRegisteredQueryField,
								"queryField3",
								"queryField4",
								"queryField5",
								"queryField6",
								"queryField7",
								"queryField8",
								"queryField9",
								"queryField10",
								"queryField11",
								"queryField12",
								"queryField13",
								"queryField14",
								"queryField15",
								"queryField16",
								"queryField17",
								"queryField18",
								"queryField19",
								"queryField20",
								"queryField21",
							},
						},
					},
				},
				token: ValidToken,
				mockTokens: map[string]bool{
					ValidToken: true,
				},
				deviceId: RegisteredDeviceId,
				deviceRequest: datafetcher.SensorDataRequest{
					Hardware: datafetcher.Hardware{
						QueryFields: []string{
							RegisteredQueryField,
							AnotherRegisteredQueryField,
							"queryField3",
							"queryField4",
							"queryField5",
							"queryField6",
							"queryField7",
							"queryField8",
							"queryField9",
							"queryField10",
							"queryField11",
							"queryField12",
							"queryField13",
							"queryField14",
							"queryField15",
							"queryField16",
							"queryField17",
							"queryField18",
							"queryField19",
							"queryField20",
							"queryField21",
						},
					},
					TimeFrame: datafetcher.TimeFrame{
						Start: RelativeStart,
					},
				},
			},
		},

		"admin user can get all device queryfields": {
			want: []datafetcher.SensorData{
				{
					DeviceID:  RegisteredDeviceId,
					Timestamp: InsideTimeRange,
					SensorData: map[string]float64{
						RegisteredQueryField:        23,
						AnotherRegisteredQueryField: 80,
					},
				},
			},
			gsdt: GetSensorDataTest{
				wantErr:    false,
				wantStatus: http.StatusOK,
				mockAuthStore: authstore.Schema{
					UserInfo: []authstore.UserInfo{
						{
							Username: RegisteredUsername,
							Company:  AnotherRegisteredCompany,
							Role:     int(authstore.Admin),
							Password: RegisteredPassword,
							Network:  AnotherRegisteredNetwork,
						},
					},
					UserTokens: []authstore.UserToken{
						{Username: RegisteredUsername, Token: ValidToken},
					},
				},
				mockDataFetcher: []datafetcher.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField:        23,
							AnotherRegisteredQueryField: 80,
						},
					},
				},
				mockDeviceInfo: deviceinfo.Schema{
					DeviceCompanies: []deviceinfo.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []deviceinfo.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []deviceinfo.DeviceToQueryFields{
						{
							DeviceId: RegisteredDeviceId,
							QueryFields: []string{
								RegisteredQueryField,
								AnotherRegisteredQueryField,
							},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
				token:    ValidToken,
				deviceId: RegisteredDeviceId,
				deviceRequest: datafetcher.SensorDataRequest{
					Hardware: datafetcher.Hardware{QueryFields: []string{"all"}},
					TimeFrame: datafetcher.TimeFrame{
						Start: RelativeStart,
					},
				},
			},
		},

		"network user can get all device queryfields": {
			want: []datafetcher.SensorData{
				{
					DeviceID:  RegisteredDeviceId,
					Timestamp: InsideTimeRange,
					SensorData: map[string]float64{
						RegisteredQueryField:        23,
						AnotherRegisteredQueryField: 80,
					},
				},
			},
			gsdt: GetSensorDataTest{wantErr: false,
				wantStatus: http.StatusOK,
				mockAuthStore: authstore.Schema{
					UserInfo: []authstore.UserInfo{
						{
							Username: RegisteredUsername,
							Company:  AnotherRegisteredCompany,
							Role:     int(authstore.NetworkUser),
							Password: RegisteredPassword,
							Network:  RegisteredNetwork,
						},
					},
					UserTokens: []authstore.UserToken{
						{Username: RegisteredUsername, Token: ValidToken},
					},
				},
				mockDataFetcher: []datafetcher.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField:        23,
							AnotherRegisteredQueryField: 80,
						},
					},
				},
				mockDeviceInfo: deviceinfo.Schema{
					DeviceCompanies: []deviceinfo.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []deviceinfo.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []deviceinfo.DeviceToQueryFields{
						{
							DeviceId: RegisteredDeviceId,
							QueryFields: []string{
								RegisteredQueryField,
								AnotherRegisteredQueryField,
							},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
				token:    ValidToken,
				deviceId: RegisteredDeviceId,
				deviceRequest: datafetcher.SensorDataRequest{
					Hardware: datafetcher.Hardware{QueryFields: []string{"all"}},
					TimeFrame: datafetcher.TimeFrame{
						Start: RelativeStart,
					},
				},
			},
		},

		"user can get all device queryfields": {
			want: []datafetcher.SensorData{
				{
					DeviceID:  RegisteredDeviceId,
					Timestamp: InsideTimeRange,
					SensorData: map[string]float64{
						RegisteredQueryField:        23,
						AnotherRegisteredQueryField: 80,
					},
				},
			},
			gsdt: GetSensorDataTest{wantErr: false,
				wantStatus: http.StatusOK,
				mockAuthStore: authstore.Schema{
					UserInfo: []authstore.UserInfo{
						{
							Username: RegisteredUsername,
							Company:  RegisteredCompany,
							Role:     int(authstore.User),
							Password: RegisteredPassword,
							Network:  RegisteredNetwork,
						},
					},
					UserTokens: []authstore.UserToken{
						{Username: RegisteredUsername, Token: ValidToken},
					},
				},
				mockDataFetcher: []datafetcher.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField:        23,
							AnotherRegisteredQueryField: 80,
						},
					},
				},
				mockDeviceInfo: deviceinfo.Schema{
					DeviceCompanies: []deviceinfo.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []deviceinfo.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []deviceinfo.DeviceToQueryFields{
						{
							DeviceId: RegisteredDeviceId,
							QueryFields: []string{
								RegisteredQueryField,
								AnotherRegisteredQueryField,
							},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
				token:    ValidToken,
				deviceId: RegisteredDeviceId,
				deviceRequest: datafetcher.SensorDataRequest{
					Hardware: datafetcher.Hardware{QueryFields: []string{"all"}},
					TimeFrame: datafetcher.TimeFrame{
						Start: RelativeStart,
					},
				},
			},
		},

		"unknown token": {
			want: nil,
			gsdt: GetSensorDataTest{wantErr: true,
				wantStatus: http.StatusUnauthorized,
				token:      InvalidToken,
				deviceId:   RegisteredDeviceId,
				deviceRequest: datafetcher.SensorDataRequest{
					Hardware: datafetcher.Hardware{QueryFields: []string{RegisteredQueryField}},
					TimeFrame: datafetcher.TimeFrame{
						Start: RelativeStart,
					},
				},
			},
		},

		"start time in future": {
			want: nil,
			gsdt: GetSensorDataTest{wantErr: true,
				wantStatus: http.StatusBadRequest,
				token:      ValidToken,
				deviceId:   RegisteredDeviceId,
				deviceRequest: datafetcher.SensorDataRequest{
					Hardware: datafetcher.Hardware{QueryFields: []string{RegisteredQueryField}},
					TimeFrame: datafetcher.TimeFrame{
						Start: FutureStart,
						Stop:  Stop,
					},
				},
				mockAuthStore: authstore.Schema{
					UserInfo: []authstore.UserInfo{
						{
							Username: RegisteredUsername,
							Company:  RegisteredCompany,
							Role:     int(authstore.User),
							Password: RegisteredPassword,
							Network:  RegisteredNetwork,
						},
					},
					UserTokens: []authstore.UserToken{
						{Username: RegisteredUsername, Token: ValidToken},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
		},

		"start time greater than stop time": {
			want: nil,
			gsdt: GetSensorDataTest{wantErr: true,
				wantStatus: http.StatusBadRequest,
				token:      ValidToken,
				deviceId:   RegisteredDeviceId,
				deviceRequest: datafetcher.SensorDataRequest{
					Hardware: datafetcher.Hardware{QueryFields: []string{RegisteredQueryField}},
					TimeFrame: datafetcher.TimeFrame{
						Start: StartGreaterThanStop,
						Stop:  Stop,
					},
				},
				mockAuthStore: authstore.Schema{
					UserInfo: []authstore.UserInfo{
						{
							Username: RegisteredUsername,
							Company:  RegisteredCompany,
							Role:     int(authstore.User),
							Password: RegisteredPassword,
							Network:  RegisteredNetwork,
						},
					},
					UserTokens: []authstore.UserToken{
						{Username: RegisteredUsername, Token: ValidToken},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
		},

		"stop time in future": {
			want: []datafetcher.SensorData{
				{
					DeviceID:  RegisteredDeviceId,
					Timestamp: InsideTimeRange,
					SensorData: map[string]float64{
						RegisteredQueryField: 23,
					},
				},
				{
					DeviceID:  RegisteredDeviceId,
					Timestamp: AlsoInsideTimeRange,
					SensorData: map[string]float64{
						RegisteredQueryField: 25,
					},
				},
			},
			gsdt: GetSensorDataTest{wantErr: false,
				wantStatus: http.StatusOK,
				mockAuthStore: authstore.Schema{
					UserInfo: []authstore.UserInfo{
						{
							Username: RegisteredUsername,
							Company:  RegisteredCompany,
							Role:     int(authstore.User),
							Password: RegisteredPassword,
							Network:  RegisteredNetwork,
						},
					},
					UserTokens: []authstore.UserToken{
						{Username: RegisteredUsername, Token: ValidToken},
					},
				},
				mockDataFetcher: []datafetcher.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField: 23,
						},
					},
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: AlsoInsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField: 25,
						},
					},
				},
				mockDeviceInfo: deviceinfo.Schema{
					DeviceCompanies: []deviceinfo.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []deviceinfo.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []deviceinfo.DeviceToQueryFields{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
				token:    ValidToken,
				deviceId: RegisteredDeviceId,
				deviceRequest: datafetcher.SensorDataRequest{
					Hardware: datafetcher.Hardware{QueryFields: []string{RegisteredQueryField}},
					TimeFrame: datafetcher.TimeFrame{
						Start: Start,
						Stop:  StopInFuture,
					},
				},
			},
		},

		"relative start time more than 90 days in the past": {
			want: nil,
			gsdt: GetSensorDataTest{wantErr: true,
				wantStatus: http.StatusBadRequest,
				token:      ValidToken,
				mockAuthStore: authstore.Schema{
					UserInfo: []authstore.UserInfo{
						{
							Username: RegisteredUsername,
							Company:  RegisteredCompany,
							Role:     int(authstore.User),
							Password: RegisteredPassword,
							Network:  RegisteredNetwork,
						},
					},
					UserTokens: []authstore.UserToken{
						{Username: RegisteredUsername, Token: ValidToken},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
				deviceId: RegisteredDeviceId,
				deviceRequest: datafetcher.SensorDataRequest{
					Hardware: datafetcher.Hardware{QueryFields: []string{RegisteredQueryField}},
					TimeFrame: datafetcher.TimeFrame{
						Start: RelativeMoreThanNinetyDays,
					},
				},
			},
		},

		"start time more than 90 days in the past": {
			want: nil,
			gsdt: GetSensorDataTest{wantErr: true,
				wantStatus: http.StatusBadRequest,
				token:      ValidToken,
				mockAuthStore: authstore.Schema{
					UserInfo: []authstore.UserInfo{
						{
							Username: RegisteredUsername,
							Company:  RegisteredCompany,
							Role:     int(authstore.User),
							Password: RegisteredPassword,
							Network:  RegisteredNetwork,
						},
					},
					UserTokens: []authstore.UserToken{
						{Username: RegisteredUsername, Token: ValidToken},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
				deviceId: RegisteredDeviceId,
				deviceRequest: datafetcher.SensorDataRequest{
					Hardware: datafetcher.Hardware{QueryFields: []string{RegisteredQueryField}},
					TimeFrame: datafetcher.TimeFrame{
						Start: MoreThanNinetyDays,
						Stop:  Stop,
					},
				},
			},
		},

		"get multiple data points within time range": {
			want: []datafetcher.SensorData{
				{
					DeviceID:  RegisteredDeviceId,
					Timestamp: InsideTimeRange,
					SensorData: map[string]float64{
						RegisteredQueryField: 23,
					},
				},
				{
					DeviceID:  RegisteredDeviceId,
					Timestamp: AlsoInsideTimeRange,
					SensorData: map[string]float64{
						RegisteredQueryField: 25,
					},
				},
			},
			gsdt: GetSensorDataTest{wantErr: false,
				wantStatus: http.StatusOK,
				mockAuthStore: authstore.Schema{
					UserInfo: []authstore.UserInfo{
						{
							Username: RegisteredUsername,
							Company:  RegisteredCompany,
							Role:     int(authstore.User),
							Password: RegisteredPassword,
							Network:  RegisteredNetwork,
						},
					},
					UserTokens: []authstore.UserToken{
						{Username: RegisteredUsername, Token: ValidToken},
					},
				},
				mockDataFetcher: []datafetcher.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField: 23,
						},
					},
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: AlsoInsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField: 25,
						},
					},
				},
				mockDeviceInfo: deviceinfo.Schema{
					DeviceCompanies: []deviceinfo.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []deviceinfo.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []deviceinfo.DeviceToQueryFields{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
				token:    ValidToken,
				deviceId: RegisteredDeviceId,
				deviceRequest: datafetcher.SensorDataRequest{
					Hardware: datafetcher.Hardware{QueryFields: []string{RegisteredQueryField}},
					TimeFrame: datafetcher.TimeFrame{
						Start: RelativeStart,
					},
				},
			},
		},

		"get multiple queryfields' data": {
			want: []datafetcher.SensorData{
				{
					DeviceID:  RegisteredDeviceId,
					Timestamp: InsideTimeRange,
					SensorData: map[string]float64{
						RegisteredQueryField:        23,
						AnotherRegisteredQueryField: 80,
					},
				},
				{
					DeviceID:  RegisteredDeviceId,
					Timestamp: AlsoInsideTimeRange,
					SensorData: map[string]float64{
						RegisteredQueryField:        25,
						AnotherRegisteredQueryField: 70,
					},
				},
			},
			gsdt: GetSensorDataTest{wantErr: false,
				wantStatus: http.StatusOK,
				mockAuthStore: authstore.Schema{
					UserInfo: []authstore.UserInfo{
						{
							Username: RegisteredUsername,
							Company:  RegisteredCompany,
							Role:     int(authstore.User),
							Password: RegisteredPassword,
							Network:  RegisteredNetwork,
						},
					},
					UserTokens: []authstore.UserToken{
						{Username: RegisteredUsername, Token: ValidToken},
					},
				},
				mockDataFetcher: []datafetcher.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField:        23,
							AnotherRegisteredQueryField: 80,
						},
					},
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: AlsoInsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField:        25,
							AnotherRegisteredQueryField: 70,
						},
					},
				},
				mockDeviceInfo: deviceinfo.Schema{
					DeviceCompanies: []deviceinfo.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []deviceinfo.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []deviceinfo.DeviceToQueryFields{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
				token:    ValidToken,
				deviceId: RegisteredDeviceId,
				deviceRequest: datafetcher.SensorDataRequest{
					Hardware: datafetcher.Hardware{QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField}},
					TimeFrame: datafetcher.TimeFrame{
						Start: RelativeStart,
					},
				},
			},
		},

		"exclude data points outside requested time range": {
			want: []datafetcher.SensorData{
				{
					DeviceID:  RegisteredDeviceId,
					Timestamp: InsideTimeRange,
					SensorData: map[string]float64{
						RegisteredQueryField: 23,
					},
				},
				{
					DeviceID:  RegisteredDeviceId,
					Timestamp: AlsoInsideTimeRange,
					SensorData: map[string]float64{
						RegisteredQueryField: 25,
					},
				},
			},
			gsdt: GetSensorDataTest{wantErr: false,
				wantStatus: http.StatusOK,
				mockAuthStore: authstore.Schema{
					UserInfo: []authstore.UserInfo{
						{
							Username: RegisteredUsername,
							Company:  RegisteredCompany,
							Role:     int(authstore.User),
							Password: RegisteredPassword,
							Network:  RegisteredNetwork,
						},
					},
					UserTokens: []authstore.UserToken{
						{Username: RegisteredUsername, Token: ValidToken},
					},
				},
				mockDataFetcher: []datafetcher.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: OutsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField: 22,
						},
					},
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField: 23,
						},
					},
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: AlsoInsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField: 25,
						},
					},
				},
				mockDeviceInfo: deviceinfo.Schema{
					DeviceCompanies: []deviceinfo.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []deviceinfo.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []deviceinfo.DeviceToQueryFields{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
				token:    ValidToken,
				deviceId: RegisteredDeviceId,
				deviceRequest: datafetcher.SensorDataRequest{
					Hardware: datafetcher.Hardware{QueryFields: []string{RegisteredQueryField}},
					TimeFrame: datafetcher.TimeFrame{
						Start: RelativeStart,
					},
				},
			},
		},

		"exclude data points outside requested time range, using relative start time": {
			want: []datafetcher.SensorData{
				{
					DeviceID:  RegisteredDeviceId,
					Timestamp: InsideTimeRange,
					SensorData: map[string]float64{
						RegisteredQueryField: 23,
					},
				},
				{
					DeviceID:  RegisteredDeviceId,
					Timestamp: AlsoInsideTimeRange,
					SensorData: map[string]float64{
						RegisteredQueryField: 25,
					},
				},
			},
			gsdt: GetSensorDataTest{wantErr: false,
				wantStatus: http.StatusOK,
				mockAuthStore: authstore.Schema{
					UserInfo: []authstore.UserInfo{
						{
							Username: RegisteredUsername,
							Company:  RegisteredCompany,
							Role:     int(authstore.User),
							Password: RegisteredPassword,
							Network:  RegisteredNetwork,
						},
					},
					UserTokens: []authstore.UserToken{
						{Username: RegisteredUsername, Token: ValidToken},
					},
				},
				mockDataFetcher: []datafetcher.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: OutsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField: 22,
						},
					},
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField: 23,
						},
					},
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: AlsoInsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField: 25,
						},
					},
				},
				mockDeviceInfo: deviceinfo.Schema{
					DeviceCompanies: []deviceinfo.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []deviceinfo.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []deviceinfo.DeviceToQueryFields{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
				token:    ValidToken,
				deviceId: RegisteredDeviceId,
				deviceRequest: datafetcher.SensorDataRequest{
					Hardware: datafetcher.Hardware{QueryFields: []string{RegisteredQueryField}},
					TimeFrame: datafetcher.TimeFrame{
						Start: RelativeStart,
					},
				},
			},
		},

		"no data in requested range": {
			want: nil,
			gsdt: GetSensorDataTest{wantErr: true,
				wantStatus: http.StatusNoContent,
				mockAuthStore: authstore.Schema{
					UserInfo: []authstore.UserInfo{
						{
							Username: RegisteredUsername,
							Company:  RegisteredCompany,
							Role:     int(authstore.User),
							Password: RegisteredPassword,
							Network:  RegisteredNetwork,
						},
					},
					UserTokens: []authstore.UserToken{
						{Username: RegisteredUsername, Token: ValidToken},
					},
				},
				mockDataFetcher: []datafetcher.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField: 23,
						},
					},
				},
				mockDeviceInfo: deviceinfo.Schema{
					DeviceCompanies: []deviceinfo.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []deviceinfo.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []deviceinfo.DeviceToQueryFields{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
				token:    ValidToken,
				deviceId: RegisteredDeviceId,
				deviceRequest: datafetcher.SensorDataRequest{
					Hardware: datafetcher.Hardware{QueryFields: []string{RegisteredQueryField}},
					TimeFrame: datafetcher.TimeFrame{
						Start: "-1h",
					},
				},
			},
		},

		"device doesnt exist": {
			want: nil,
			gsdt: GetSensorDataTest{wantErr: true,
				wantStatus: http.StatusNotFound,
				mockAuthStore: authstore.Schema{
					UserInfo: []authstore.UserInfo{
						{
							Username: RegisteredUsername,
							Company:  RegisteredCompany,
							Role:     int(authstore.User),
							Password: RegisteredPassword,
							Network:  RegisteredNetwork,
						},
					},
					UserTokens: []authstore.UserToken{
						{Username: RegisteredUsername, Token: ValidToken},
					},
				},
				mockDataFetcher: []datafetcher.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField: 23,
						},
					},
				},
				mockDeviceInfo: deviceinfo.Schema{
					DeviceCompanies: []deviceinfo.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []deviceinfo.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []deviceinfo.DeviceToQueryFields{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
				token:    ValidToken,
				deviceId: UnregisteredDeviceId,
				deviceRequest: datafetcher.SensorDataRequest{
					Hardware: datafetcher.Hardware{QueryFields: []string{RegisteredQueryField}},
					TimeFrame: datafetcher.TimeFrame{
						Start: "-1h",
					},
				},
			},
		},

		"non admin can't request deviceid not in user company": {
			want: nil,
			gsdt: GetSensorDataTest{wantErr: true,
				wantStatus: http.StatusUnauthorized,
				mockAuthStore: authstore.Schema{
					UserInfo: []authstore.UserInfo{
						{
							Username: RegisteredUsername,
							Company:  OtherCompanyThanDevice,
							Role:     int(authstore.User),
							Password: RegisteredPassword,
							Network:  RegisteredNetwork,
						},
					},
					UserTokens: []authstore.UserToken{
						{Username: RegisteredUsername, Token: ValidToken},
					},
				},
				mockDataFetcher: []datafetcher.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField: 23,
						},
					},
				},
				mockDeviceInfo: deviceinfo.Schema{
					DeviceCompanies: []deviceinfo.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []deviceinfo.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []deviceinfo.DeviceToQueryFields{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
				token:    ValidToken,
				deviceId: RegisteredDeviceId,
				deviceRequest: datafetcher.SensorDataRequest{
					Hardware: datafetcher.Hardware{QueryFields: []string{RegisteredQueryField}},
					TimeFrame: datafetcher.TimeFrame{
						Start: RelativeStart,
					},
				},
			},
		},

		"admin user can request deviceid not in user company": {
			want: []datafetcher.SensorData{
				{
					DeviceID:  RegisteredDeviceId,
					Timestamp: InsideTimeRange,
					SensorData: map[string]float64{
						RegisteredQueryField: 23,
					},
				},
			},
			gsdt: GetSensorDataTest{wantErr: false,
				wantStatus: http.StatusOK,
				mockAuthStore: authstore.Schema{
					UserInfo: []authstore.UserInfo{
						{
							Username: RegisteredUsername,
							Company:  OtherCompanyThanDevice,
							Role:     int(authstore.Admin),
							Password: RegisteredPassword,
							Network:  RegisteredNetwork,
						},
					},
					UserTokens: []authstore.UserToken{
						{Username: RegisteredUsername, Token: ValidToken},
					},
				},
				mockDataFetcher: []datafetcher.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField: 23,
						},
					},
				},
				mockDeviceInfo: deviceinfo.Schema{
					DeviceCompanies: []deviceinfo.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []deviceinfo.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []deviceinfo.DeviceToQueryFields{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
				token:    ValidToken,
				deviceId: RegisteredDeviceId,
				deviceRequest: datafetcher.SensorDataRequest{
					Hardware: datafetcher.Hardware{QueryFields: []string{RegisteredQueryField}},
					TimeFrame: datafetcher.TimeFrame{
						Start: RelativeStart,
					},
				},
			},
		},
	}

	db, err := miniredis.Run()
	require.Nil(t, err)
	defer db.Close()
	humaTest := setupHuma(t)
	var closeFunc CloseFunc
	var qp string
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			closeFunc = setupGetSensorDataTest(t, tc.gsdt, db)
			defer closeFunc()
			qp = makeQueryParams(tc.gsdt.deviceRequest)
			route := "/device/" + tc.gsdt.deviceId + "/sensordata" + qp
			resp := humaTest.Get(route,
				fmt.Sprintf(`Authorization: Bearer %s`, tc.gsdt.token))
			if resp.Code != tc.gsdt.wantStatus {
				t.Fatalf("wantStatus: %d, response status: %d", tc.gsdt.wantStatus, resp.Code)
			}
			defer resp.Result().Body.Close()
			if !tc.gsdt.wantErr {
				var dd []datafetcher.SensorData
				body := resp.Body.Bytes()
				err = json.Unmarshal(body, &dd)
				require.Nil(t, err)
				if diff := cmp.Diff(tc.want, dd, cmpOpts...); diff != "" {
					t.Fatalf("response mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func setupHuma(t *testing.T) humatest.TestAPI {
	config := localhuma.Config(localhuma.Production)
	router := mux.NewRouter()
	humaApi := humamux.New(router, config)
	localhuma.SetupApi(humaApi, a)
	return humatest.Wrap(t, humaApi)
}

func makeQueryParams(dr datafetcher.SensorDataRequest) string {
	b := strings.Builder{}
	start := url.QueryEscape(dr.Start)
	fmt.Fprintf(&b, "?start=%s", start)
	if dr.Stop != "" {
		stop := url.QueryEscape(dr.Stop)
		fmt.Fprintf(&b, "&stop=%s", stop)
	}
	for _, q := range dr.QueryFields {
		fmt.Fprintf(&b, "&queryField=%s", q)
	}
	fmt.Fprintf(&b, "&timezone-return=%s", dr.Timezone)
	return b.String()
}

func TestGetQueryFields(t *testing.T) {
	tests := map[string]struct {
		wantErr        bool
		wantStatus     int
		deviceId       string
		want           deviceinfo.QueryFields
		mockTokens     map[string]bool
		mockAuthStore  authstore.Schema
		mockDeviceInfo deviceinfo.Schema
		token          string
	}{

		"successfully get queryfields": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			deviceId:   RegisteredDeviceId,
			want: deviceinfo.QueryFields{
				DeviceId:    RegisteredDeviceId,
				QueryFields: append(deviceinfo.GeneralQueryFields, RegisteredQueryField),
			},
			mockAuthStore: authstore.Schema{
				UserInfo: []authstore.UserInfo{
					{
						Username: RegisteredUsername,
						Company:  RegisteredCompany,
						Role:     int(authstore.User),
						Password: RegisteredPassword,
						Network:  RegisteredNetwork,
					},
				},
				UserTokens: []authstore.UserToken{
					{Username: RegisteredUsername, Token: ValidToken},
				},
			},
			mockTokens: map[string]bool{
				ValidToken: true,
			},
			token: ValidToken,
			mockDeviceInfo: deviceinfo.Schema{
				DeviceCompanies: []deviceinfo.DeviceToCompany{
					{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
				},
				DeviceNetworks: []deviceinfo.DeviceToNetwork{
					{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
				},
				DeviceToQF: []deviceinfo.DeviceToQueryFields{
					{
						DeviceId:    RegisteredDeviceId,
						QueryFields: []string{RegisteredQueryField},
					},
				},
			},
		},

		"regular user cannot request queryfields for device from other company": {
			wantErr:    true,
			wantStatus: http.StatusUnauthorized,
			deviceId:   RegisteredDeviceId,
			want:       deviceinfo.QueryFields{},
			mockAuthStore: authstore.Schema{
				UserInfo: []authstore.UserInfo{
					{
						Username: RegisteredUsername,
						Company:  OtherCompanyThanDevice,
						Role:     int(authstore.User),
						Password: RegisteredPassword,
						Network:  RegisteredNetwork,
					},
				},
				UserTokens: []authstore.UserToken{
					{Username: RegisteredUsername, Token: ValidToken},
				},
			},
			mockTokens: map[string]bool{
				ValidToken: true,
			},
			token: ValidToken,
			mockDeviceInfo: deviceinfo.Schema{
				DeviceCompanies: []deviceinfo.DeviceToCompany{
					{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
				},
				DeviceNetworks: []deviceinfo.DeviceToNetwork{
					{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
				},
				DeviceToQF: []deviceinfo.DeviceToQueryFields{
					{
						DeviceId:    RegisteredDeviceId,
						QueryFields: []string{RegisteredQueryField},
					},
				},
			},
		},

		"admin user can request queryfields for device from other company": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			deviceId:   RegisteredDeviceId,
			want: deviceinfo.QueryFields{
				DeviceId:    RegisteredDeviceId,
				QueryFields: append(deviceinfo.GeneralQueryFields, RegisteredQueryField),
			},
			mockAuthStore: authstore.Schema{
				UserInfo: []authstore.UserInfo{
					{
						Username: RegisteredUsername,
						Company:  OtherCompanyThanDevice,
						Role:     int(authstore.Admin),
						Password: RegisteredPassword,
						Network:  RegisteredNetwork,
					},
				},
				UserTokens: []authstore.UserToken{
					{Username: RegisteredUsername, Token: ValidToken},
				},
			},
			mockTokens: map[string]bool{
				ValidToken: true,
			},
			token: ValidToken,
			mockDeviceInfo: deviceinfo.Schema{
				DeviceCompanies: []deviceinfo.DeviceToCompany{
					{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
				},
				DeviceNetworks: []deviceinfo.DeviceToNetwork{
					{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
				},
				DeviceToQF: []deviceinfo.DeviceToQueryFields{
					{
						DeviceId:    RegisteredDeviceId,
						QueryFields: []string{RegisteredQueryField},
					},
				},
			},
		},

		"unknown token": {
			wantErr:    true,
			wantStatus: http.StatusUnauthorized,
			token:      InvalidToken,
			want:       deviceinfo.QueryFields{},
			deviceId:   RegisteredDeviceId,
		},
	}

	db, err := miniredis.Run()
	require.Nil(t, err)
	defer db.Close()
	humaTest := setupHuma(t)
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			a.TokenProvider = &tokenprovider.MockTokenProvider{
				Tokens:    tc.mockTokens,
				Increment: len(tc.mockTokens),
			}
			defer a.TokenProvider.Close()
			testingRedis, err := redis.NewTestingRedis(db.Addr())
			require.Nil(t, err)
			defer testingRedis.Close()
			a.DeviceInfo = testingRedis
			a.AuthStore = testingRedis
			err = testingRedis.PrepareDeviceInfo(tc.mockDeviceInfo)
			require.Nil(t, err)
			err = testingRedis.PrepareAuthStore(tc.mockAuthStore)
			require.Nil(t, err)
			route := "/device/" + tc.deviceId + "/queryfields"
			resp := humaTest.Get(route,
				fmt.Sprintf(`Authorization: Bearer %s`, tc.token))
			if resp.Code != tc.wantStatus {
				t.Fatalf("wantStatus: %d, response status: %d", tc.wantStatus, resp.Code)
			}
			defer resp.Result().Body.Close()
			if !tc.wantErr {
				var qf deviceinfo.QueryFields
				body := resp.Body.Bytes()
				err = json.Unmarshal(body, &qf)
				require.Nil(t, err)
				if diff := cmp.Diff(tc.want, qf, cmpOpts...); diff != "" {
					t.Fatalf("response mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestGetDeviceIds(t *testing.T) {
	tests := map[string]struct {
		wantErr        bool
		wantStatus     int
		want           []string
		mockAuthStore  authstore.Schema
		mockDeviceInfo deviceinfo.Schema
		mockTokens     map[string]bool
		token          string
	}{

		"user gets deviceIds only in company, in network": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want:       []string{RegisteredDeviceId},
			mockAuthStore: authstore.Schema{
				UserInfo: []authstore.UserInfo{
					{
						Username: RegisteredUsername,
						Company:  RegisteredCompany,
						Role:     int(authstore.User),
						Password: RegisteredPassword,
						Network:  RegisteredNetwork,
					},
				},
				UserTokens: []authstore.UserToken{
					{Username: RegisteredUsername, Token: ValidToken},
				},
			},
			mockDeviceInfo: deviceinfo.Schema{
				DeviceCompanies: []deviceinfo.DeviceToCompany{
					{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					{DeviceId: AnotherRegisteredDeviceId, Company: AnotherRegisteredCompany},
				},
				DeviceNetworks: []deviceinfo.DeviceToNetwork{
					{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					{DeviceId: AnotherRegisteredDeviceId, Network: AnotherRegisteredNetwork},
				},
				DeviceToQF: []deviceinfo.DeviceToQueryFields{
					{
						DeviceId:    RegisteredDeviceId,
						QueryFields: []string{RegisteredQueryField},
					},
					{
						DeviceId:    AnotherRegisteredDeviceId,
						QueryFields: []string{AnotherRegisteredQueryField},
					},
				},
			},
			mockTokens: map[string]bool{
				ValidToken: true,
			},
			token: ValidToken,
		},

		"network user gets deviceIds in network": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want:       []string{RegisteredDeviceId, AnotherRegisteredDeviceId},
			mockAuthStore: authstore.Schema{
				UserInfo: []authstore.UserInfo{
					{
						Username: RegisteredUsername,
						Company:  RegisteredCompany,
						Role:     int(authstore.NetworkUser),
						Password: RegisteredPassword,
						Network:  RegisteredNetwork,
					},
				},
				UserTokens: []authstore.UserToken{
					{Username: RegisteredUsername, Token: ValidToken},
				},
			},
			mockDeviceInfo: deviceinfo.Schema{
				DeviceCompanies: []deviceinfo.DeviceToCompany{
					{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					{DeviceId: AnotherRegisteredDeviceId, Company: AnotherRegisteredCompany},
				},
				DeviceNetworks: []deviceinfo.DeviceToNetwork{
					{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					{DeviceId: AnotherRegisteredDeviceId, Network: RegisteredNetwork},
				},
				DeviceToQF: []deviceinfo.DeviceToQueryFields{
					{
						DeviceId:    RegisteredDeviceId,
						QueryFields: []string{RegisteredQueryField},
					},
					{
						DeviceId:    AnotherRegisteredDeviceId,
						QueryFields: []string{AnotherRegisteredQueryField},
					},
				},
			},
			mockTokens: map[string]bool{
				ValidToken: true,
			},
			token: ValidToken,
		},

		"admin gets all deviceIds": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want:       []string{RegisteredDeviceId, AnotherRegisteredDeviceId},
			mockAuthStore: authstore.Schema{
				UserInfo: []authstore.UserInfo{
					{
						Username: RegisteredUsername,
						Company:  RegisteredCompany,
						Role:     int(authstore.Admin),
						Password: RegisteredPassword,
						Network:  RegisteredNetwork,
					},
				},
				UserTokens: []authstore.UserToken{
					{Username: RegisteredUsername, Token: ValidToken},
				},
			},
			mockDeviceInfo: deviceinfo.Schema{
				DeviceCompanies: []deviceinfo.DeviceToCompany{
					{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					{DeviceId: AnotherRegisteredDeviceId, Company: AnotherRegisteredCompany},
				},
				DeviceNetworks: []deviceinfo.DeviceToNetwork{
					{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					{DeviceId: AnotherRegisteredDeviceId, Network: AnotherRegisteredNetwork},
				},
				DeviceToQF: []deviceinfo.DeviceToQueryFields{
					{
						DeviceId:    RegisteredDeviceId,
						QueryFields: []string{RegisteredQueryField},
					},
					{
						DeviceId:    AnotherRegisteredDeviceId,
						QueryFields: []string{AnotherRegisteredQueryField},
					},
				},
			},
			mockTokens: map[string]bool{
				ValidToken: true,
			},
			token: ValidToken,
		},
	}
	db, err := miniredis.Run()
	require.Nil(t, err)
	defer db.Close()
	humaTest := setupHuma(t)
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			a.TokenProvider = &tokenprovider.MockTokenProvider{
				Tokens:    tc.mockTokens,
				Increment: len(tc.mockTokens),
			}
			defer a.TokenProvider.Close()
			testingRedis, err := redis.NewTestingRedis(db.Addr())
			require.Nil(t, err)
			defer testingRedis.Close()
			a.DeviceInfo = testingRedis
			a.AuthStore = testingRedis
			err = testingRedis.PrepareDeviceInfo(tc.mockDeviceInfo)
			require.Nil(t, err)
			err = testingRedis.PrepareAuthStore(tc.mockAuthStore)
			require.Nil(t, err)
			route := "/device/ids"
			resp := humaTest.Get(route,
				fmt.Sprintf(`Authorization: Bearer %s`, tc.token))
			if resp.Code != tc.wantStatus {
				t.Fatalf("wantStatus: %d, response status: %d", tc.wantStatus, resp.Code)
			}
			defer resp.Result().Body.Close()
			if !tc.wantErr {
				var dr []string
				body := resp.Body.Bytes()
				err = json.Unmarshal(body, &dr)
				require.Nil(t, err)
				if diff := cmp.Diff(tc.want, dr, cmpOpts...); diff != "" {
					t.Fatalf("response mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestCheckOlderThanNinetyDays(t *testing.T) {
	tests := map[string]struct {
		input string
		want  bool
	}{
		"older using days suffix":    {input: "-91d", want: true},
		"older using minutes suffix": {input: "-129601m", want: true},
		"older using seconds suffix": {input: "-7776001s", want: true},
		"older using hours suffix":   {input: "-2161h", want: true},
		"older using months suffix":  {input: "-4mo", want: true},
		"newer using days suffix":    {input: "-90d", want: false},
		"newer using minutes suffix": {input: "-129600m", want: false},
		"newer using seconds suffix": {input: "-7776000s", want: false},
		"newer using hours suffix":   {input: "-2160h", want: false},
		"newer using months suffix":  {input: "-3mo", want: false},
		"invalid suffix":             {input: "-1du", want: true},
		"invalid prefix":             {input: "1s", want: true},
		"no number":                  {input: "rtyu", want: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			older := api.CheckOlderThanNinetyDays(tc.input)
			if tc.want != older {
				t.Fatalf("want: %v, got: %v", tc.want, older)
			}
		})
	}
}

func TestGetDataBoundary(t *testing.T) {
	tests := map[string]struct {
		wantErr         bool
		wantStatus      int
		mockDataFetcher []datafetcher.SensorData
		want            datafetcher.DataBoundary
		mockAuthStore   authstore.Schema
		mockDeviceInfo  deviceinfo.Schema
		mockTokens      map[string]bool
		token           string
		deviceId        string
	}{

		"user get deviceid data boundary": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: datafetcher.DataBoundary{
				DeviceId: RegisteredDeviceId,
				Start:    InsideTimeRange,
				Stop:     AlsoInsideTimeRange,
			},
			mockAuthStore: authstore.Schema{
				UserInfo: []authstore.UserInfo{
					{
						Username: RegisteredUsername,
						Company:  RegisteredCompany,
						Role:     int(authstore.User),
						Password: RegisteredPassword,
						Network:  RegisteredNetwork,
					},
				},
				UserTokens: []authstore.UserToken{
					{Username: RegisteredUsername, Token: ValidToken},
				},
			},
			mockDataFetcher: []datafetcher.SensorData{
				{
					DeviceID:  RegisteredDeviceId,
					Timestamp: InsideTimeRange,
					SensorData: map[string]float64{
						RegisteredQueryField: 23,
						"batv":               3.4,
					},
				},
				{
					DeviceID:  RegisteredDeviceId,
					Timestamp: AlsoInsideTimeRange,
					SensorData: map[string]float64{
						RegisteredQueryField: 23,
						"batv":               3.4,
					},
				},
			},
			mockDeviceInfo: deviceinfo.Schema{
				DeviceCompanies: []deviceinfo.DeviceToCompany{
					{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
				},
				DeviceNetworks: []deviceinfo.DeviceToNetwork{
					{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
				},
				DeviceToQF: []deviceinfo.DeviceToQueryFields{
					{
						DeviceId:    RegisteredDeviceId,
						QueryFields: []string{RegisteredQueryField},
					},
				},
			},
			mockTokens: map[string]bool{
				ValidToken: true,
			},
			token:    ValidToken,
			deviceId: RegisteredDeviceId,
		},

		"network user get deviceid data boundary": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: datafetcher.DataBoundary{
				DeviceId: RegisteredDeviceId,
				Start:    InsideTimeRange,
				Stop:     AlsoInsideTimeRange,
			},
			mockAuthStore: authstore.Schema{
				UserInfo: []authstore.UserInfo{
					{
						Username: RegisteredUsername,
						Company:  AnotherRegisteredCompany,
						Role:     int(authstore.NetworkUser),
						Password: RegisteredPassword,
						Network:  RegisteredNetwork,
					},
				},
				UserTokens: []authstore.UserToken{
					{Username: RegisteredUsername, Token: ValidToken},
				},
			},
			mockDataFetcher: []datafetcher.SensorData{
				{
					DeviceID:  RegisteredDeviceId,
					Timestamp: InsideTimeRange,
					SensorData: map[string]float64{
						RegisteredQueryField: 23,
						"batv":               3.4,
					},
				},
				{
					DeviceID:  RegisteredDeviceId,
					Timestamp: AlsoInsideTimeRange,
					SensorData: map[string]float64{
						RegisteredQueryField: 23,
						"batv":               3.4,
					},
				},
			},
			mockDeviceInfo: deviceinfo.Schema{
				DeviceCompanies: []deviceinfo.DeviceToCompany{
					{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
				},
				DeviceNetworks: []deviceinfo.DeviceToNetwork{
					{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
				},
				DeviceToQF: []deviceinfo.DeviceToQueryFields{
					{
						DeviceId:    RegisteredDeviceId,
						QueryFields: []string{RegisteredQueryField},
					},
				},
			},
			mockTokens: map[string]bool{
				ValidToken: true,
			},
			token:    ValidToken,
			deviceId: RegisteredDeviceId,
		},

		"admin user get deviceid data boundary": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: datafetcher.DataBoundary{
				DeviceId: RegisteredDeviceId,
				Start:    InsideTimeRange,
				Stop:     AlsoInsideTimeRange,
			},
			mockAuthStore: authstore.Schema{
				UserInfo: []authstore.UserInfo{
					{
						Username: RegisteredUsername,
						Company:  AnotherRegisteredCompany,
						Role:     int(authstore.Admin),
						Password: RegisteredPassword,
						Network:  AnotherRegisteredNetwork,
					},
				},
				UserTokens: []authstore.UserToken{
					{Username: RegisteredUsername, Token: ValidToken},
				},
			},
			mockDataFetcher: []datafetcher.SensorData{
				{
					DeviceID:  RegisteredDeviceId,
					Timestamp: InsideTimeRange,
					SensorData: map[string]float64{
						RegisteredQueryField: 23,
						"batv":               3.4,
					},
				},
				{
					DeviceID:  RegisteredDeviceId,
					Timestamp: AlsoInsideTimeRange,
					SensorData: map[string]float64{
						RegisteredQueryField: 23,
						"batv":               3.4,
					},
				},
			},
			mockDeviceInfo: deviceinfo.Schema{
				DeviceCompanies: []deviceinfo.DeviceToCompany{
					{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
				},
				DeviceNetworks: []deviceinfo.DeviceToNetwork{
					{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
				},
				DeviceToQF: []deviceinfo.DeviceToQueryFields{
					{
						DeviceId:    RegisteredDeviceId,
						QueryFields: []string{RegisteredQueryField},
					},
				},
			},
			mockTokens: map[string]bool{
				ValidToken: true,
			},
			token:    ValidToken,
			deviceId: RegisteredDeviceId,
		},

		"unknown token": {
			wantErr:    true,
			wantStatus: http.StatusUnauthorized,
			token:      InvalidToken,
			want:       datafetcher.DataBoundary{},
			deviceId:   RegisteredDeviceId,
		},
	}
	db, err := miniredis.Run()
	require.Nil(t, err)
	defer db.Close()
	humaTest := setupHuma(t)
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			a.TokenProvider = &tokenprovider.MockTokenProvider{
				Tokens:    tc.mockTokens,
				Increment: len(tc.mockTokens),
			}
			defer a.TokenProvider.Close()
			testingRedis, err := redis.NewTestingRedis(db.Addr())
			require.Nil(t, err)
			defer testingRedis.Close()
			a.DeviceInfo = testingRedis
			a.AuthStore = testingRedis
			a.DataFetcher, err = datafetcher.NewTestingInflux("../config.yml")
			require.Nil(t, err)
			defer a.DataFetcher.Close()
			err = a.DataFetcher.PrepareDb(&tc.mockDeviceInfo, tc.mockDataFetcher)
			require.Nil(t, err)
			err = testingRedis.PrepareDeviceInfo(tc.mockDeviceInfo)
			require.Nil(t, err)
			err = testingRedis.PrepareAuthStore(tc.mockAuthStore)
			require.Nil(t, err)
			route := "/device/" + tc.deviceId + "/databoundary"
			resp := humaTest.Get(route,
				fmt.Sprintf(`Authorization: Bearer %s`, tc.token))
			if resp.Code != tc.wantStatus {
				t.Fatalf("wantStatus: %d, response status: %d", tc.wantStatus, resp.Code)
			}
			defer resp.Result().Body.Close()
			if !tc.wantErr {
				var got datafetcher.DataBoundary
				body := resp.Body.Bytes()
				err = json.Unmarshal(body, &got)
				require.Nil(t, err)
				if diff := cmp.Diff(tc.want, got, cmpOpts...); diff != "" {
					t.Fatalf("response mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

type GetSensorDataTest struct {
	wantErr         bool
	wantStatus      int
	mockDataFetcher []datafetcher.SensorData
	mockAuthStore   authstore.Schema
	mockDeviceInfo  deviceinfo.Schema
	mockTokens      map[string]bool
	token           string
	deviceId        string
	deviceRequest   datafetcher.SensorDataRequest
	batchRequests   datafetcher.BatchSensorDataRequest
}

func TestCsvGetSensorData(t *testing.T) {
	tests := map[string]struct {
		want string
		gsdt GetSensorDataTest
	}{

		"single deviceid, single queryfield": {
			want: fmt.Sprintf(",%s\n%s\n%s,%s\n",
				RegisteredQueryField, RegisteredDeviceId,
				InsideTimeRange.Format(time.RFC3339), "23.000"),
			gsdt: GetSensorDataTest{wantErr: false,
				wantStatus: http.StatusOK,
				mockAuthStore: authstore.Schema{
					UserInfo: []authstore.UserInfo{
						{
							Username: RegisteredUsername,
							Company:  RegisteredCompany,
							Role:     int(authstore.User),
							Password: RegisteredPassword,
							Network:  RegisteredNetwork,
						},
					},
					UserTokens: []authstore.UserToken{
						{Username: RegisteredUsername, Token: ValidToken},
					},
				},
				mockDataFetcher: []datafetcher.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField: 23,
						},
					},
				},
				mockDeviceInfo: deviceinfo.Schema{
					DeviceCompanies: []deviceinfo.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []deviceinfo.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []deviceinfo.DeviceToQueryFields{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
				token:    ValidToken,
				deviceId: RegisteredDeviceId,
				deviceRequest: datafetcher.SensorDataRequest{
					Hardware: datafetcher.Hardware{QueryFields: []string{RegisteredQueryField}},
					TimeFrame: datafetcher.TimeFrame{
						Start: RelativeStart,
					},
				},
			},
		},

		"single deviceid, single queryfield with specific timezone": {
			want: fmt.Sprintf(",%s\n%s\n%s,%s\n",
				RegisteredQueryField, RegisteredDeviceId,
				InsideTimeRange.Local().Format(time.RFC3339), "23.000"),
			gsdt: GetSensorDataTest{wantErr: false,
				wantStatus: http.StatusOK,
				mockAuthStore: authstore.Schema{
					UserInfo: []authstore.UserInfo{
						{
							Username: RegisteredUsername,
							Company:  RegisteredCompany,
							Role:     int(authstore.User),
							Password: RegisteredPassword,
							Network:  RegisteredNetwork,
						},
					},
					UserTokens: []authstore.UserToken{
						{Username: RegisteredUsername, Token: ValidToken},
					},
				},
				mockDataFetcher: []datafetcher.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField: 23,
						},
					},
				},
				mockDeviceInfo: deviceinfo.Schema{
					DeviceCompanies: []deviceinfo.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []deviceinfo.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []deviceinfo.DeviceToQueryFields{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
				token:    ValidToken,
				deviceId: RegisteredDeviceId,
				deviceRequest: datafetcher.SensorDataRequest{
					Hardware: datafetcher.Hardware{QueryFields: []string{RegisteredQueryField}},
					TimeFrame: datafetcher.TimeFrame{
						Start:    RelativeStart,
						Timezone: datafetcher.Timezone{Timezone: "Africa/Johannesburg"},
					},
				},
			},
		},

		"single deviceid, multiple queryfield": {
			want: fmt.Sprintf(",%s,%s\n%s\n%s,%s,%s\n",
				AnotherRegisteredQueryField, RegisteredQueryField, RegisteredDeviceId,
				InsideTimeRange.Format(time.RFC3339), "80.000", "23.000"),
			gsdt: GetSensorDataTest{wantErr: false,
				wantStatus: http.StatusOK,
				mockAuthStore: authstore.Schema{
					UserInfo: []authstore.UserInfo{
						{
							Username: RegisteredUsername,
							Company:  RegisteredCompany,
							Role:     int(authstore.User),
							Password: RegisteredPassword,
							Network:  RegisteredNetwork,
						},
					},
					UserTokens: []authstore.UserToken{
						{Username: RegisteredUsername, Token: ValidToken},
					},
				},
				mockDataFetcher: []datafetcher.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField:        23,
							AnotherRegisteredQueryField: 80,
						},
					},
				},
				mockDeviceInfo: deviceinfo.Schema{
					DeviceCompanies: []deviceinfo.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []deviceinfo.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []deviceinfo.DeviceToQueryFields{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
				token:    ValidToken,
				deviceId: RegisteredDeviceId,
				deviceRequest: datafetcher.SensorDataRequest{
					Hardware: datafetcher.Hardware{QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField}},
					TimeFrame: datafetcher.TimeFrame{
						Start: RelativeStart,
					},
				},
			},
		},

		"single deviceid, multiple queryfield and timestamp": {
			want: fmt.Sprintf(",%s,%s\n%s\n%s,%s,%s\n%s,%s,%s\n",
				AnotherRegisteredQueryField, RegisteredQueryField, RegisteredDeviceId,
				InsideTimeRange.Format(time.RFC3339), "80.000", "23.000",
				AlsoInsideTimeRange.Format(time.RFC3339), "81.000", "25.000"),
			gsdt: GetSensorDataTest{wantErr: false,
				wantStatus: http.StatusOK,
				mockAuthStore: authstore.Schema{
					UserInfo: []authstore.UserInfo{
						{
							Username: RegisteredUsername,
							Company:  RegisteredCompany,
							Role:     int(authstore.User),
							Password: RegisteredPassword,
							Network:  RegisteredNetwork,
						},
					},
					UserTokens: []authstore.UserToken{
						{Username: RegisteredUsername, Token: ValidToken},
					},
				},
				mockDataFetcher: []datafetcher.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField:        23,
							AnotherRegisteredQueryField: 80,
						},
					},
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: AlsoInsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField:        25,
							AnotherRegisteredQueryField: 81,
						},
					},
				},
				mockDeviceInfo: deviceinfo.Schema{
					DeviceCompanies: []deviceinfo.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []deviceinfo.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []deviceinfo.DeviceToQueryFields{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
				token:    ValidToken,
				deviceId: RegisteredDeviceId,
				deviceRequest: datafetcher.SensorDataRequest{
					Hardware: datafetcher.Hardware{QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField}},
					TimeFrame: datafetcher.TimeFrame{
						Start: RelativeStart,
					},
				},
			},
		},

		"single deviceid, multiple queryfield and seperate timestamp": {
			want: fmt.Sprintf(",%s,%s\n%s\n%s,%s,\n%s,,%s\n",
				AnotherRegisteredQueryField, RegisteredQueryField, RegisteredDeviceId,
				InsideTimeRange.Format(time.RFC3339), "80.000",
				AlsoInsideTimeRange.Format(time.RFC3339), "25.000"),
			gsdt: GetSensorDataTest{wantErr: false,
				wantStatus: http.StatusOK,
				mockAuthStore: authstore.Schema{
					UserInfo: []authstore.UserInfo{
						{
							Username: RegisteredUsername,
							Company:  RegisteredCompany,
							Role:     int(authstore.User),
							Password: RegisteredPassword,
							Network:  RegisteredNetwork,
						},
					},
					UserTokens: []authstore.UserToken{
						{Username: RegisteredUsername, Token: ValidToken},
					},
				},
				mockDataFetcher: []datafetcher.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							AnotherRegisteredQueryField: 80,
						},
					},
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: AlsoInsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField: 25,
						},
					},
				},
				mockDeviceInfo: deviceinfo.Schema{
					DeviceCompanies: []deviceinfo.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []deviceinfo.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []deviceinfo.DeviceToQueryFields{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
				token:    ValidToken,
				deviceId: RegisteredDeviceId,
				deviceRequest: datafetcher.SensorDataRequest{
					Hardware: datafetcher.Hardware{QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField}},
					TimeFrame: datafetcher.TimeFrame{
						Start: RelativeStart,
					},
				},
			},
		},
	}

	db, err := miniredis.Run()
	require.Nil(t, err)
	defer db.Close()
	humaTest := setupHuma(t)
	var qp string
	var close CloseFunc
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			close = setupGetSensorDataTest(t, tc.gsdt, db)
			defer close()
			qp = makeQueryParams(tc.gsdt.deviceRequest)
			route := "/device/" + tc.gsdt.deviceId + "/sensordata" + qp
			resp := humaTest.Get(route,
				fmt.Sprintf(`Authorization: Bearer %s`, tc.gsdt.token),
				"Accept: text/csv",
			)
			if resp.Code != tc.gsdt.wantStatus {
				t.Fatalf("wantStatus: %d, response status: %d", tc.gsdt.wantStatus, resp.Code)
			}
			contentType := resp.Header().Get("Content-Type")
			if contentType != "text/csv" {
				t.Fatalf("response content-type not csv: %s", contentType)
			}
			defer resp.Result().Body.Close()
			if !tc.gsdt.wantErr {
				body := resp.Body.String()
				if diff := cmp.Diff(tc.want, body, cmpOpts...); diff != "" {
					t.Fatalf("response mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

type CloseFunc func()

func setupGetSensorDataTest(
	t *testing.T, tc GetSensorDataTest, db *miniredis.Miniredis) CloseFunc {
	a.TokenProvider = &tokenprovider.MockTokenProvider{
		Tokens:    tc.mockTokens,
		Increment: len(tc.mockTokens),
	}
	testingRedis, err := redis.NewTestingRedis(db.Addr())
	require.Nil(t, err)
	a.DeviceInfo = testingRedis
	a.AuthStore = testingRedis
	a.DataFetcher, err = datafetcher.NewTestingInflux("../config.yml")
	require.Nil(t, err)
	err = a.DataFetcher.PrepareDb(&tc.mockDeviceInfo, tc.mockDataFetcher)
	require.Nil(t, err)
	err = testingRedis.PrepareDeviceInfo(tc.mockDeviceInfo)
	require.Nil(t, err)
	err = testingRedis.PrepareAuthStore(tc.mockAuthStore)
	require.Nil(t, err)
	return func() {
		err = a.TokenProvider.Close()
		if err != nil {
			t.Logf("tokenprovider close: %v", err)
		}
		err = testingRedis.Close()
		if err != nil {
			t.Logf("testingRedis close: %v", err)
		}
		err = a.DataFetcher.Close()
		if err != nil {
			t.Logf("datafetcher close: %v", err)
		}
	}
}
