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
	"github.com/danielgtaylor/huma/v2"
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
const RegisteredNetwork = "Datafarm"
const AnotherRegisteredNetwork = "Datafarm2"
const RegisteredDeviceId = "device1"
const AnotherRegisteredDeviceId = "device2"
const InvalidDeviceId = "!+)$"
const RegisteredQueryField = "temperature"
const AnotherRegisteredQueryField = "humidity"
const RegisteredSensor = "weather-sensor"
const ValidToken = "someToken0"
const InvalidToken = "invalidToken0"
const RelativeStart = "-6h"
const RelativeMoreThanThirtyOneDays = "-32d"

var MoreThanThirtyOneDays = time.Now().Add(-31 * 24 * time.Hour).Format(time.RFC3339)
var Start = time.Now().Add(-24 * time.Hour).Format(time.RFC3339)
var StartGreaterThanStop = time.Now().Add(1 * time.Hour).Format(time.RFC3339)
var FutureStart = time.Now().Add(1 * time.Hour).Format(time.RFC3339)
var Stop = time.Now().Format(time.RFC3339)
var StopInFuture = time.Now().Add(24 * time.Hour).Format(time.RFC3339)
var OutsideTimeRange = time.Now().Add(-25 * time.Hour)
var InsideTimeRange = time.Now().Add(-2 * time.Hour)
var AlsoInsideTimeRange = time.Now().Add(-1 * time.Hour)
var RegisteredCompanyDevices = []string{RegisteredDeviceId}
var a = &api.Api{}

func TestLogin(t *testing.T) {
	tests := map[string]struct {
		wantErr            bool
		wantStatus         int
		username, password string
		mockAuthStore      authstore.Schema
		mockDeviceInfo     deviceinfo.Schema
	}{

		"successfully login": {
			wantErr:    false,
			wantStatus: http.StatusOK,
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
			}
		})
	}
}

func TestGetDeviceData(t *testing.T) {
	tests := map[string]struct {
		wantErr               bool
		wantStatus            int
		mockDataFetcher, want []datafetcher.DeviceData
		mockAuthStore         authstore.Schema
		mockDeviceInfo        deviceinfo.Schema
		mockTokens            map[string]bool
		token                 string
		deviceId              string
		deviceRequest         datafetcher.DeviceDataRequest
	}{

		"successfully get deviceid data": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: []datafetcher.DeviceData{
				{
					DeviceID:  RegisteredDeviceId,
					Timestamp: InsideTimeRange,
					SensorData: map[string]float64{
						RegisteredQueryField: 23,
					},
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
			mockDataFetcher: []datafetcher.DeviceData{
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
			deviceRequest: datafetcher.DeviceDataRequest{
				QueryFields: []string{RegisteredQueryField},
				Start:       RelativeStart,
			},
		},

		"admin user can get all device queryfields": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: []datafetcher.DeviceData{
				{
					DeviceID:  RegisteredDeviceId,
					Timestamp: InsideTimeRange,
					SensorData: map[string]float64{
						RegisteredQueryField:        23,
						AnotherRegisteredQueryField: 80,
					},
				},
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
			mockDataFetcher: []datafetcher.DeviceData{
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
			deviceRequest: datafetcher.DeviceDataRequest{
				QueryFields: []string{"all"},
				Start:       RelativeStart,
			},
		},

		"network user can get all device queryfields": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: []datafetcher.DeviceData{
				{
					DeviceID:  RegisteredDeviceId,
					Timestamp: InsideTimeRange,
					SensorData: map[string]float64{
						RegisteredQueryField:        23,
						AnotherRegisteredQueryField: 80,
					},
				},
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
			mockDataFetcher: []datafetcher.DeviceData{
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
			deviceRequest: datafetcher.DeviceDataRequest{
				QueryFields: []string{"all"},
				Start:       RelativeStart,
			},
		},

		"user can get all device queryfields": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: []datafetcher.DeviceData{
				{
					DeviceID:  RegisteredDeviceId,
					Timestamp: InsideTimeRange,
					SensorData: map[string]float64{
						RegisteredQueryField:        23,
						AnotherRegisteredQueryField: 80,
					},
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
			mockDataFetcher: []datafetcher.DeviceData{
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
			deviceRequest: datafetcher.DeviceDataRequest{
				QueryFields: []string{"all"},
				Start:       RelativeStart,
			},
		},

		"unknown token": {
			wantErr:    true,
			wantStatus: http.StatusUnauthorized,
			token:      InvalidToken,
			want:       nil,
			deviceId:   RegisteredDeviceId,
			deviceRequest: datafetcher.DeviceDataRequest{
				QueryFields: []string{RegisteredQueryField},
				Start:       RelativeStart,
			},
		},

		"start time in future": {
			wantErr:    true,
			wantStatus: http.StatusBadRequest,
			token:      ValidToken,
			want:       nil,
			deviceId:   RegisteredDeviceId,
			deviceRequest: datafetcher.DeviceDataRequest{
				QueryFields: []string{RegisteredQueryField},
				Start:       FutureStart,
				Stop:        Stop,
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

		"start time greater than stop time": {
			wantErr:    true,
			wantStatus: http.StatusBadRequest,
			token:      ValidToken,
			want:       nil,
			deviceId:   RegisteredDeviceId,
			deviceRequest: datafetcher.DeviceDataRequest{
				QueryFields: []string{RegisteredQueryField},
				Start:       StartGreaterThanStop,
				Stop:        Stop,
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

		"stop time in future": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: []datafetcher.DeviceData{
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
			mockDataFetcher: []datafetcher.DeviceData{
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
			deviceRequest: datafetcher.DeviceDataRequest{
				QueryFields: []string{RegisteredQueryField},
				Start:       Start,
				Stop:        StopInFuture,
			},
		},

		"relative start time more than 31 days in the past": {
			wantErr:    true,
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
			want:     nil,
			deviceId: RegisteredDeviceId,
			deviceRequest: datafetcher.DeviceDataRequest{
				QueryFields: []string{RegisteredQueryField},
				Start:       RelativeMoreThanThirtyOneDays,
			},
		},

		"start time more than 31 days in the past": {
			wantErr:    true,
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
			want:     nil,
			deviceId: RegisteredDeviceId,
			deviceRequest: datafetcher.DeviceDataRequest{
				QueryFields: []string{RegisteredQueryField},
				Start:       MoreThanThirtyOneDays,
				Stop:        Stop,
			},
		},

		"get multiple data points within time range": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: []datafetcher.DeviceData{
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
			mockDataFetcher: []datafetcher.DeviceData{
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
			deviceRequest: datafetcher.DeviceDataRequest{
				QueryFields: []string{RegisteredQueryField},
				Start:       RelativeStart,
			},
		},

		"get multiple queryfields' data": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: []datafetcher.DeviceData{
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
			mockDataFetcher: []datafetcher.DeviceData{
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
			deviceRequest: datafetcher.DeviceDataRequest{
				QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField},
				Start:       RelativeStart,
			},
		},

		"exclude data points outside requested time range": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: []datafetcher.DeviceData{
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
			mockDataFetcher: []datafetcher.DeviceData{
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
			deviceRequest: datafetcher.DeviceDataRequest{
				QueryFields: []string{RegisteredQueryField},
				Start:       RelativeStart,
			},
		},

		"exclude data points outside requested time range, using relative start time": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: []datafetcher.DeviceData{
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
			mockDataFetcher: []datafetcher.DeviceData{
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
			deviceRequest: datafetcher.DeviceDataRequest{
				QueryFields: []string{RegisteredQueryField},
				Start:       RelativeStart,
			},
		},

		"no data in requested range": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want:       nil,
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
			mockDataFetcher: []datafetcher.DeviceData{
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
			deviceRequest: datafetcher.DeviceDataRequest{
				QueryFields: []string{RegisteredQueryField},
				Start:       "-1h",
			},
		},

		"non admin can't request deviceid not in user company": {
			wantErr:    true,
			wantStatus: http.StatusUnauthorized,
			want:       nil,
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
			mockDataFetcher: []datafetcher.DeviceData{
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
			deviceRequest: datafetcher.DeviceDataRequest{
				QueryFields: []string{RegisteredQueryField},
				Start:       RelativeStart,
			},
		},

		"admin user can request deviceid not in user company": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: []datafetcher.DeviceData{
				{
					DeviceID:  RegisteredDeviceId,
					Timestamp: InsideTimeRange,
					SensorData: map[string]float64{
						RegisteredQueryField: 23,
					},
				},
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
			mockDataFetcher: []datafetcher.DeviceData{
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
			deviceRequest: datafetcher.DeviceDataRequest{
				QueryFields: []string{RegisteredQueryField},
				Start:       RelativeStart,
			},
		},
	}

	db, err := miniredis.Run()
	require.Nil(t, err)
	defer db.Close()
	humaTest := setupHuma(t)
	var qp string
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
			qp = makeQueryParams(tc.deviceRequest)
			route := "/device/" + tc.deviceId + "/sensordata" + qp
			resp := humaTest.Get(route,
				fmt.Sprintf(`Authorization: Bearer %s`, tc.token))
			if resp.Code != tc.wantStatus {
				t.Fatalf("wantStatus: %d, response status: %d", tc.wantStatus, resp.Code)
			}
			defer resp.Result().Body.Close()
			if !tc.wantErr {
				var dd []datafetcher.DeviceData
				body := resp.Body.Bytes()
				err = json.Unmarshal(body, &dd)
				require.Nil(t, err)
				if diff := cmp.Diff(tc.want, dd); diff != "" {
					t.Fatalf("response mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func setupHuma(t *testing.T) humatest.TestAPI {
	router := mux.NewRouter()
	config := huma.DefaultConfig("DataFarm SensorData API", "1.0.0")
	humaApiMux := humamux.New(router, config)
	humaTest := humatest.Wrap(t, humaApiMux)
	localhuma.RegisterHumaOperations(humaTest,
		a.RateLimit, a.VerifyToken, a.GetDeviceData, a.BatchGetDeviceData,
		a.Login, a.GetQueryFields, a.BatchGetQueryFields, a.GetDeviceIds)
	return humaTest
}

func makeQueryParams(dr datafetcher.DeviceDataRequest) string {
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
				if diff := cmp.Diff(tc.want, qf); diff != "" {
					t.Fatalf("response mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestBatchGetDeviceData(t *testing.T) {
	tests := map[string]struct {
		wantErr         bool
		wantStatus      int
		mockDataFetcher []datafetcher.DeviceData
		want            datafetcher.BatchDeviceDataResponse
		mockAuthStore   authstore.Schema
		mockDeviceInfo  deviceinfo.Schema
		mockTokens      map[string]bool
		token           string
		deviceRequests  []datafetcher.DeviceDataRequest
	}{

		"get multiple deviceIds' data": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: datafetcher.BatchDeviceDataResponse{
				Errors: []datafetcher.DeviceDataError{},
				Results: []datafetcher.DeviceData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField:        23,
							AnotherRegisteredQueryField: 80,
						},
					},
					{
						DeviceID:  AnotherRegisteredDeviceId,
						Timestamp: AlsoInsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField:        25,
							AnotherRegisteredQueryField: 70,
						},
					},
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
			mockDataFetcher: []datafetcher.DeviceData{
				{
					DeviceID:  RegisteredDeviceId,
					Timestamp: InsideTimeRange,
					SensorData: map[string]float64{
						RegisteredQueryField:        23,
						AnotherRegisteredQueryField: 80,
					},
				},
				{
					DeviceID:  AnotherRegisteredDeviceId,
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
					{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
				},
				DeviceNetworks: []deviceinfo.DeviceToNetwork{
					{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					{DeviceId: AnotherRegisteredDeviceId, Network: RegisteredNetwork},
				},
				DeviceToQF: []deviceinfo.DeviceToQueryFields{
					{
						DeviceId:    RegisteredDeviceId,
						QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField},
					},
					{
						DeviceId:    AnotherRegisteredDeviceId,
						QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField},
					},
				},
			},
			mockTokens: map[string]bool{
				ValidToken: true,
			},
			token: ValidToken,
			deviceRequests: []datafetcher.DeviceDataRequest{
				{
					DeviceId:    RegisteredDeviceId,
					QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField},
					Start:       RelativeStart,
				},
				{
					DeviceId:    AnotherRegisteredDeviceId,
					QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField},
					Start:       RelativeStart,
				},
			},
		},

		"admin user can get device data from any company": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: datafetcher.BatchDeviceDataResponse{
				Errors: []datafetcher.DeviceDataError{},
				Results: []datafetcher.DeviceData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField:        23,
							AnotherRegisteredQueryField: 80,
						},
					},
					{
						DeviceID:  AnotherRegisteredDeviceId,
						Timestamp: AlsoInsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField:        25,
							AnotherRegisteredQueryField: 70,
						},
					},
				},
			},
			mockAuthStore: authstore.Schema{
				UserInfo: []authstore.UserInfo{
					{
						Username: RegisteredUsername,
						Company:  AnotherRegisteredCompany,
						Role:     int(authstore.Admin),
						Password: RegisteredPassword,
						Network:  RegisteredNetwork,
					},
				},
				UserTokens: []authstore.UserToken{
					{Username: RegisteredUsername, Token: ValidToken},
				},
			},
			mockDataFetcher: []datafetcher.DeviceData{
				{
					DeviceID:  RegisteredDeviceId,
					Timestamp: InsideTimeRange,
					SensorData: map[string]float64{
						RegisteredQueryField:        23,
						AnotherRegisteredQueryField: 80,
					},
				},
				{
					DeviceID:  AnotherRegisteredDeviceId,
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
					{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
				},
				DeviceNetworks: []deviceinfo.DeviceToNetwork{
					{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					{DeviceId: AnotherRegisteredDeviceId, Network: RegisteredNetwork},
				},
				DeviceToQF: []deviceinfo.DeviceToQueryFields{
					{
						DeviceId:    RegisteredDeviceId,
						QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField},
					},
					{
						DeviceId:    AnotherRegisteredDeviceId,
						QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField},
					},
				},
			},
			mockTokens: map[string]bool{
				ValidToken: true,
			},
			token: ValidToken,
			deviceRequests: []datafetcher.DeviceDataRequest{
				{
					DeviceId:    RegisteredDeviceId,
					QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField},
					Start:       RelativeStart,
				},
				{
					DeviceId:    AnotherRegisteredDeviceId,
					QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField},
					Start:       RelativeStart,
				},
			},
		},

		"admin user can get device data from any network": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: datafetcher.BatchDeviceDataResponse{
				Errors: []datafetcher.DeviceDataError{},
				Results: []datafetcher.DeviceData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField:        23,
							AnotherRegisteredQueryField: 80,
						},
					},
					{
						DeviceID:  AnotherRegisteredDeviceId,
						Timestamp: AlsoInsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField:        25,
							AnotherRegisteredQueryField: 70,
						},
					},
				},
			},
			mockAuthStore: authstore.Schema{
				UserInfo: []authstore.UserInfo{
					{
						Username: RegisteredUsername,
						Company:  AnotherRegisteredCompany,
						Network:  AnotherRegisteredNetwork,
						Role:     int(authstore.Admin),
						Password: RegisteredPassword,
					},
				},
				UserTokens: []authstore.UserToken{
					{Username: RegisteredUsername, Token: ValidToken},
				},
			},
			mockDataFetcher: []datafetcher.DeviceData{
				{
					DeviceID:  RegisteredDeviceId,
					Timestamp: InsideTimeRange,
					SensorData: map[string]float64{
						RegisteredQueryField:        23,
						AnotherRegisteredQueryField: 80,
					},
				},
				{
					DeviceID:  AnotherRegisteredDeviceId,
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
					{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
				},
				DeviceNetworks: []deviceinfo.DeviceToNetwork{
					{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					{DeviceId: AnotherRegisteredDeviceId, Network: RegisteredNetwork},
				},
				DeviceToQF: []deviceinfo.DeviceToQueryFields{
					{
						DeviceId:    RegisteredDeviceId,
						QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField},
					},
					{
						DeviceId:    AnotherRegisteredDeviceId,
						QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField},
					},
				},
			},
			mockTokens: map[string]bool{
				ValidToken: true,
			},
			token: ValidToken,
			deviceRequests: []datafetcher.DeviceDataRequest{
				{
					DeviceId:    RegisteredDeviceId,
					QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField},
					Start:       RelativeStart,
				},
				{
					DeviceId:    AnotherRegisteredDeviceId,
					QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField},
					Start:       RelativeStart,
				},
			},
		},

		"network user can get any device data from within network": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: datafetcher.BatchDeviceDataResponse{
				Errors: []datafetcher.DeviceDataError{},
				Results: []datafetcher.DeviceData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField:        23,
							AnotherRegisteredQueryField: 80,
						},
					},
					{
						DeviceID:  AnotherRegisteredDeviceId,
						Timestamp: AlsoInsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField:        25,
							AnotherRegisteredQueryField: 70,
						},
					},
				},
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
			mockDataFetcher: []datafetcher.DeviceData{
				{
					DeviceID:  RegisteredDeviceId,
					Timestamp: InsideTimeRange,
					SensorData: map[string]float64{
						RegisteredQueryField:        23,
						AnotherRegisteredQueryField: 80,
					},
				},
				{
					DeviceID:  AnotherRegisteredDeviceId,
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
					{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
				},
				DeviceNetworks: []deviceinfo.DeviceToNetwork{
					{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					{DeviceId: AnotherRegisteredDeviceId, Network: RegisteredNetwork},
				},
				DeviceToQF: []deviceinfo.DeviceToQueryFields{
					{
						DeviceId:    RegisteredDeviceId,
						QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField},
					},
					{
						DeviceId:    AnotherRegisteredDeviceId,
						QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField},
					},
				},
			},
			mockTokens: map[string]bool{
				ValidToken: true,
			},
			token: ValidToken,
			deviceRequests: []datafetcher.DeviceDataRequest{
				{
					DeviceId:    RegisteredDeviceId,
					QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField},
					Start:       RelativeStart,
				},
				{
					DeviceId:    AnotherRegisteredDeviceId,
					QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField},
					Start:       RelativeStart,
				},
			},
		},

		"network user cant get device data from other network": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: datafetcher.BatchDeviceDataResponse{
				Errors: []datafetcher.DeviceDataError{
					{
						DeviceId: RegisteredDeviceId,
						Error:    "Unauthorized access to this device.",
					},
					{
						DeviceId: AnotherRegisteredDeviceId,
						Error:    "Unauthorized access to this device.",
					},
				},
				Results: []datafetcher.DeviceData{},
			},
			mockAuthStore: authstore.Schema{
				UserInfo: []authstore.UserInfo{
					{
						Username: RegisteredUsername,
						Company:  RegisteredCompany,
						Network:  AnotherRegisteredNetwork,
						Role:     int(authstore.NetworkUser),
						Password: RegisteredPassword,
					},
				},
				UserTokens: []authstore.UserToken{
					{Username: RegisteredUsername, Token: ValidToken},
				},
			},
			mockDataFetcher: []datafetcher.DeviceData{
				{
					DeviceID:  RegisteredDeviceId,
					Timestamp: InsideTimeRange,
					SensorData: map[string]float64{
						RegisteredQueryField:        23,
						AnotherRegisteredQueryField: 80,
					},
				},
				{
					DeviceID:  AnotherRegisteredDeviceId,
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
					{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
				},
				DeviceNetworks: []deviceinfo.DeviceToNetwork{
					{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					{DeviceId: AnotherRegisteredDeviceId, Network: RegisteredNetwork},
				},
				DeviceToQF: []deviceinfo.DeviceToQueryFields{
					{
						DeviceId:    RegisteredDeviceId,
						QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField},
					},
					{
						DeviceId:    AnotherRegisteredDeviceId,
						QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField},
					},
				},
			},
			mockTokens: map[string]bool{
				ValidToken: true,
			},
			token: ValidToken,
			deviceRequests: []datafetcher.DeviceDataRequest{
				{
					DeviceId:    RegisteredDeviceId,
					QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField},
					Start:       RelativeStart,
				},
				{
					DeviceId:    AnotherRegisteredDeviceId,
					QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField},
					Start:       RelativeStart,
				},
			},
		},

		"user cant get device data from other company": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: datafetcher.BatchDeviceDataResponse{
				Errors: []datafetcher.DeviceDataError{
					{
						DeviceId: RegisteredDeviceId,
						Error:    "Unauthorized access to this device.",
					},
					{
						DeviceId: AnotherRegisteredDeviceId,
						Error:    "Unauthorized access to this device.",
					},
				},
				Results: []datafetcher.DeviceData{},
			},
			mockAuthStore: authstore.Schema{
				UserInfo: []authstore.UserInfo{
					{
						Username: RegisteredUsername,
						Company:  AnotherRegisteredCompany,
						Role:     int(authstore.User),
						Password: RegisteredPassword,
						Network:  RegisteredNetwork,
					},
				},
				UserTokens: []authstore.UserToken{
					{Username: RegisteredUsername, Token: ValidToken},
				},
			},
			mockDataFetcher: []datafetcher.DeviceData{
				{
					DeviceID:  RegisteredDeviceId,
					Timestamp: InsideTimeRange,
					SensorData: map[string]float64{
						RegisteredQueryField:        23,
						AnotherRegisteredQueryField: 80,
					},
				},
				{
					DeviceID:  AnotherRegisteredDeviceId,
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
					{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
				},
				DeviceNetworks: []deviceinfo.DeviceToNetwork{
					{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					{DeviceId: AnotherRegisteredDeviceId, Network: RegisteredNetwork},
				},
				DeviceToQF: []deviceinfo.DeviceToQueryFields{
					{
						DeviceId:    RegisteredDeviceId,
						QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField},
					},
					{
						DeviceId:    AnotherRegisteredDeviceId,
						QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField},
					},
				},
			},
			mockTokens: map[string]bool{
				ValidToken: true,
			},
			token: ValidToken,
			deviceRequests: []datafetcher.DeviceDataRequest{
				{
					DeviceId:    RegisteredDeviceId,
					QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField},
					Start:       RelativeStart,
				},
				{
					DeviceId:    AnotherRegisteredDeviceId,
					QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField},
					Start:       RelativeStart,
				},
			},
		},

		"one successful request, one error": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: datafetcher.BatchDeviceDataResponse{
				Errors: []datafetcher.DeviceDataError{
					{
						DeviceId: AnotherRegisteredDeviceId,
						Error:    "Unauthorized access to this device.",
					},
				},
				Results: []datafetcher.DeviceData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField:        23,
							AnotherRegisteredQueryField: 80,
						},
					},
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
			mockDataFetcher: []datafetcher.DeviceData{
				{
					DeviceID:  RegisteredDeviceId,
					Timestamp: InsideTimeRange,
					SensorData: map[string]float64{
						RegisteredQueryField:        23,
						AnotherRegisteredQueryField: 80,
					},
				},
				{
					DeviceID:  AnotherRegisteredDeviceId,
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
					{DeviceId: AnotherRegisteredDeviceId, Company: AnotherRegisteredCompany},
				},
				DeviceNetworks: []deviceinfo.DeviceToNetwork{
					{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					{DeviceId: AnotherRegisteredDeviceId, Network: RegisteredNetwork},
				},
				DeviceToQF: []deviceinfo.DeviceToQueryFields{
					{
						DeviceId:    RegisteredDeviceId,
						QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField},
					},
					{
						DeviceId:    AnotherRegisteredDeviceId,
						QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField},
					},
				},
			},
			mockTokens: map[string]bool{
				ValidToken: true,
			},
			token: ValidToken,
			deviceRequests: []datafetcher.DeviceDataRequest{
				{
					DeviceId:    RegisteredDeviceId,
					QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField},
					Start:       RelativeStart,
				},
				{
					DeviceId:    AnotherRegisteredDeviceId,
					QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField},
					Start:       RelativeStart,
				},
			},
		},

		"unknown token": {
			wantErr:    true,
			wantStatus: http.StatusUnauthorized,
			token:      InvalidToken,
			want:       datafetcher.BatchDeviceDataResponse{},
			deviceRequests: []datafetcher.DeviceDataRequest{
				{
					DeviceId:    RegisteredDeviceId,
					QueryFields: []string{RegisteredQueryField},
					Start:       RelativeStart,
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
			route := "/batch/device/sensordata"
			resp := humaTest.Post(route,
				fmt.Sprintf(`Authorization: Bearer %s`, tc.token), tc.deviceRequests)
			if resp.Code != tc.wantStatus {
				t.Fatalf("wantStatus: %d, response status: %d", tc.wantStatus, resp.Code)
			}
			defer resp.Result().Body.Close()
			if !tc.wantErr {
				var dd datafetcher.BatchDeviceDataResponse
				body := resp.Body.Bytes()
				err = json.Unmarshal(body, &dd)
				require.Nil(t, err)
				if diff := cmp.Diff(tc.want, dd); diff != "" {
					t.Fatalf("response mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestBatchGetQueryFields(t *testing.T) {
	tests := map[string]struct {
		wantErr            bool
		wantStatus         int
		want               deviceinfo.BatchQueryFieldsResponse
		mockAuthStore      authstore.Schema
		mockDeviceInfo     deviceinfo.Schema
		mockTokens         map[string]bool
		token              string
		queryFieldRequests deviceinfo.BatchQueryFieldsRequest
	}{

		"get multiple deviceIds' queryfields": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: deviceinfo.BatchQueryFieldsResponse{
				Errors: []deviceinfo.QueryFieldsError{},
				Results: []deviceinfo.QueryFields{
					{
						DeviceId:    RegisteredDeviceId,
						QueryFields: append(deviceinfo.GeneralQueryFields, RegisteredQueryField),
					},
					{
						DeviceId: AnotherRegisteredDeviceId,
						QueryFields: append(deviceinfo.GeneralQueryFields,
							AnotherRegisteredQueryField),
					},
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
			mockDeviceInfo: deviceinfo.Schema{
				DeviceCompanies: []deviceinfo.DeviceToCompany{
					{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
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
			queryFieldRequests: deviceinfo.BatchQueryFieldsRequest{
				Body: []string{
					RegisteredDeviceId,
					AnotherRegisteredDeviceId,
				},
			},
		},

		"admin user can get device queryfields from any company": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: deviceinfo.BatchQueryFieldsResponse{
				Errors: []deviceinfo.QueryFieldsError{},
				Results: []deviceinfo.QueryFields{
					{
						DeviceId:    RegisteredDeviceId,
						QueryFields: append(deviceinfo.GeneralQueryFields, RegisteredQueryField),
					},
					{
						DeviceId: AnotherRegisteredDeviceId,
						QueryFields: append(deviceinfo.GeneralQueryFields,
							AnotherRegisteredQueryField),
					},
				},
			},
			mockAuthStore: authstore.Schema{
				UserInfo: []authstore.UserInfo{
					{
						Username: RegisteredUsername,
						Company:  AnotherRegisteredCompany,
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
					{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
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
			queryFieldRequests: deviceinfo.BatchQueryFieldsRequest{
				Body: []string{
					RegisteredDeviceId,
					AnotherRegisteredDeviceId,
				},
			},
		},

		"admin user can get device queryfields from any network": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: deviceinfo.BatchQueryFieldsResponse{
				Errors: []deviceinfo.QueryFieldsError{},
				Results: []deviceinfo.QueryFields{
					{
						DeviceId:    RegisteredDeviceId,
						QueryFields: append(deviceinfo.GeneralQueryFields, RegisteredQueryField),
					},
					{
						DeviceId: AnotherRegisteredDeviceId,
						QueryFields: append(deviceinfo.GeneralQueryFields,
							AnotherRegisteredQueryField),
					},
				},
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
			mockDeviceInfo: deviceinfo.Schema{
				DeviceCompanies: []deviceinfo.DeviceToCompany{
					{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
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
			queryFieldRequests: deviceinfo.BatchQueryFieldsRequest{
				Body: []string{
					RegisteredDeviceId,
					AnotherRegisteredDeviceId,
				},
			},
		},

		"network user can get any device queryfields from within network": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: deviceinfo.BatchQueryFieldsResponse{
				Errors: []deviceinfo.QueryFieldsError{},
				Results: []deviceinfo.QueryFields{
					{
						DeviceId:    RegisteredDeviceId,
						QueryFields: append(deviceinfo.GeneralQueryFields, RegisteredQueryField),
					},
					{
						DeviceId: AnotherRegisteredDeviceId,
						QueryFields: append(deviceinfo.GeneralQueryFields,
							AnotherRegisteredQueryField),
					},
				},
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
			mockDeviceInfo: deviceinfo.Schema{
				DeviceCompanies: []deviceinfo.DeviceToCompany{
					{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
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
			queryFieldRequests: deviceinfo.BatchQueryFieldsRequest{
				Body: []string{
					RegisteredDeviceId,
					AnotherRegisteredDeviceId,
				},
			},
		},

		"network user cant get device data from other network": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: deviceinfo.BatchQueryFieldsResponse{
				Errors: []deviceinfo.QueryFieldsError{
					{
						DeviceId: RegisteredDeviceId,
						Error:    "Unauthorized access to this device.",
					},
					{
						DeviceId: AnotherRegisteredDeviceId,
						Error:    "Unauthorized access to this device.",
					},
				},
				Results: []deviceinfo.QueryFields{},
			},
			mockAuthStore: authstore.Schema{
				UserInfo: []authstore.UserInfo{
					{
						Username: RegisteredUsername,
						Company:  RegisteredCompany,
						Network:  AnotherRegisteredNetwork,
						Role:     int(authstore.NetworkUser),
						Password: RegisteredPassword,
					},
				},
				UserTokens: []authstore.UserToken{
					{Username: RegisteredUsername, Token: ValidToken},
				},
			},
			mockDeviceInfo: deviceinfo.Schema{
				DeviceCompanies: []deviceinfo.DeviceToCompany{
					{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
				},
				DeviceNetworks: []deviceinfo.DeviceToNetwork{
					{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					{DeviceId: AnotherRegisteredDeviceId, Network: RegisteredNetwork},
				},
				DeviceToQF: []deviceinfo.DeviceToQueryFields{
					{
						DeviceId:    RegisteredDeviceId,
						QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField},
					},
					{
						DeviceId:    AnotherRegisteredDeviceId,
						QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField},
					},
				},
			},
			mockTokens: map[string]bool{
				ValidToken: true,
			},
			token: ValidToken,
			queryFieldRequests: deviceinfo.BatchQueryFieldsRequest{
				Body: []string{RegisteredDeviceId, AnotherRegisteredDeviceId},
			},
		},

		"user cant get device queryfields from other company": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: deviceinfo.BatchQueryFieldsResponse{
				Errors: []deviceinfo.QueryFieldsError{
					{
						DeviceId: RegisteredDeviceId,
						Error:    "Unauthorized access to this device.",
					},
					{
						DeviceId: AnotherRegisteredDeviceId,
						Error:    "Unauthorized access to this device.",
					},
				},
				Results: []deviceinfo.QueryFields{},
			},
			mockAuthStore: authstore.Schema{
				UserInfo: []authstore.UserInfo{
					{
						Username: RegisteredUsername,
						Company:  AnotherRegisteredCompany,
						Network:  AnotherRegisteredNetwork,
						Role:     int(authstore.User),
						Password: RegisteredPassword,
					},
				},
				UserTokens: []authstore.UserToken{
					{Username: RegisteredUsername, Token: ValidToken},
				},
			},
			mockDeviceInfo: deviceinfo.Schema{
				DeviceCompanies: []deviceinfo.DeviceToCompany{
					{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
				},
				DeviceNetworks: []deviceinfo.DeviceToNetwork{
					{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					{DeviceId: AnotherRegisteredDeviceId, Network: RegisteredNetwork},
				},
				DeviceToQF: []deviceinfo.DeviceToQueryFields{
					{
						DeviceId:    RegisteredDeviceId,
						QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField},
					},
					{
						DeviceId:    AnotherRegisteredDeviceId,
						QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField},
					},
				},
			},
			mockTokens: map[string]bool{
				ValidToken: true,
			},
			token: ValidToken,
			queryFieldRequests: deviceinfo.BatchQueryFieldsRequest{
				Body: []string{RegisteredDeviceId, AnotherRegisteredDeviceId},
			},
		},

		"one accessible deviceid, one inaccessible deviceid": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: deviceinfo.BatchQueryFieldsResponse{
				Errors: []deviceinfo.QueryFieldsError{
					{
						DeviceId: AnotherRegisteredDeviceId,
						Error:    "Unauthorized access to this device.",
					},
				},
				Results: []deviceinfo.QueryFields{
					{
						DeviceId: RegisteredDeviceId,
						QueryFields: append(deviceinfo.GeneralQueryFields,
							RegisteredQueryField),
					},
				},
			},
			mockAuthStore: authstore.Schema{
				UserInfo: []authstore.UserInfo{
					{
						Username: RegisteredUsername,
						Company:  RegisteredCompany,
						Network:  RegisteredNetwork,
						Role:     int(authstore.User),
						Password: RegisteredPassword,
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
			queryFieldRequests: deviceinfo.BatchQueryFieldsRequest{
				Body: []string{RegisteredDeviceId, AnotherRegisteredDeviceId},
			},
		},

		"one valid deviceId, one invalid deviceId": {
			wantErr:    true,
			wantStatus: http.StatusUnprocessableEntity,
			want:       deviceinfo.BatchQueryFieldsResponse{},
			mockAuthStore: authstore.Schema{
				UserInfo: []authstore.UserInfo{
					{
						Username: RegisteredUsername,
						Company:  RegisteredCompany,
						Network:  RegisteredNetwork,
						Role:     int(authstore.User),
						Password: RegisteredPassword,
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
			queryFieldRequests: deviceinfo.BatchQueryFieldsRequest{
				Body: []string{RegisteredDeviceId, InvalidDeviceId},
			},
		},

		"unknown token": {
			wantErr:    true,
			wantStatus: http.StatusUnauthorized,
			token:      InvalidToken,
			want:       deviceinfo.BatchQueryFieldsResponse{},
			queryFieldRequests: deviceinfo.BatchQueryFieldsRequest{
				Body: []string{RegisteredDeviceId, AnotherRegisteredDeviceId},
			},
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
			route := "/batch/device/queryfields"
			resp := humaTest.Post(route,
				fmt.Sprintf(`Authorization: Bearer %s`, tc.token), tc.queryFieldRequests.Body)
			if resp.Code != tc.wantStatus {
				t.Fatalf("wantStatus: %d, response status: %d", tc.wantStatus, resp.Code)
			}
			defer resp.Result().Body.Close()
			if !tc.wantErr {
				var dd deviceinfo.BatchQueryFieldsResponse
				body := resp.Body.Bytes()
				err = json.Unmarshal(body, &dd)
				require.Nil(t, err)
				if diff := cmp.Diff(tc.want, dd); diff != "" {
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
				if diff := cmp.Diff(tc.want, dr); diff != "" {
					t.Fatalf("response mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestCheckOlderThanThirtyOneDays(t *testing.T) {
	tests := map[string]struct {
		input string
		want  bool
	}{
		"older using days suffix":    {input: "-32d", want: true},
		"older using minutes suffix": {input: "-44641m", want: true},
		"older using seconds suffix": {input: "-2678401s", want: true},
		"older using hours suffix":   {input: "-745h", want: true},
		"older using months suffix":  {input: "-2mo", want: true},
		"newer using days suffix":    {input: "-31d", want: false},
		"newer using minutes suffix": {input: "-44640m", want: false},
		"newer using seconds suffix": {input: "-2678400s", want: false},
		"newer using hours suffix":   {input: "-744h", want: false},
		"newer using months suffix":  {input: "-1mo", want: false},
		"invalid suffix":             {input: "-1du", want: true},
		"invalid prefix":             {input: "1s", want: true},
		"no number":                  {input: "rtyu", want: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			older := api.CheckOlderThanThirtyOneDays(tc.input)
			if tc.want != older {
				t.Fatalf("want: %v, got: %v", tc.want, older)
			}
		})
	}
}
