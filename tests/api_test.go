package tests

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
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
const OtherCompanyThanDevice = "othercompany"
const RegisteredNetwork = "Datafarm"
const UserRole = "1"
const AdminUserRole = "3"
const RegisteredDeviceId = "device1"
const RegisteredQueryField = "temperature"
const AnotherRegisteredQueryField = "humidity"
const RegisteredSensor = "weather-sensor"
const ValidToken = "someToken0"
const InvalidToken = "invalidToken0"
const RelativeStart = "-6h"

var Start = time.Now().Add(-24 * time.Hour).Format(time.RFC3339)
var StartGreaterThanStop = time.Now().Add(1 * time.Hour).Format(time.RFC3339)
var FutureStart = time.Now().Add(1 * time.Hour).Format(time.RFC3339)
var Stop = time.Now().Format(time.RFC3339)
var StopInFuture = time.Now().Add(24 * time.Hour).Format(time.RFC3339)
var OutsideTimeRange = time.Now().Add(-25 * time.Hour)
var InsideTimeRange = time.Now().Add(-2 * time.Hour)
var AlsoInsideTimeRange = time.Now().Add(-1 * time.Hour)
var a = &api.Api{
	AdminRole: AdminUserRole,
}

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
						Role:     UserRole,
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
						Role:     UserRole,
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
	_, humaApi := humatest.New(t)
	localhuma.RegisterHumaOperations(humaApi, a.VerifyToken, a.GetDeviceData, a.Login, a.GetQueryFields)
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
			resp := humaApi.Post("/login",
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
					SensorData: map[string]any{
						RegisteredQueryField: float64(23),
					},
				},
			},
			mockAuthStore: authstore.Schema{
				UserInfo: []authstore.UserInfo{
					{
						Username: RegisteredUsername,
						Company:  RegisteredCompany,
						Role:     UserRole,
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
					SensorData: map[string]any{
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
						Role:     UserRole,
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
						Role:     UserRole,
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
					SensorData: map[string]any{
						RegisteredQueryField: float64(23),
					},
				},
				{
					DeviceID:  RegisteredDeviceId,
					Timestamp: AlsoInsideTimeRange,
					SensorData: map[string]any{
						RegisteredQueryField: float64(25),
					},
				},
			},
			mockAuthStore: authstore.Schema{
				UserInfo: []authstore.UserInfo{
					{
						Username: RegisteredUsername,
						Company:  RegisteredCompany,
						Role:     UserRole,
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
					SensorData: map[string]any{
						RegisteredQueryField: 23,
					},
				},
				{
					DeviceID:  RegisteredDeviceId,
					Timestamp: AlsoInsideTimeRange,
					SensorData: map[string]any{
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

		"get multiple data points within time range": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: []datafetcher.DeviceData{
				{
					DeviceID:  RegisteredDeviceId,
					Timestamp: InsideTimeRange,
					SensorData: map[string]any{
						RegisteredQueryField: float64(23),
					},
				},
				{
					DeviceID:  RegisteredDeviceId,
					Timestamp: AlsoInsideTimeRange,
					SensorData: map[string]any{
						RegisteredQueryField: float64(25),
					},
				},
			},
			mockAuthStore: authstore.Schema{
				UserInfo: []authstore.UserInfo{
					{
						Username: RegisteredUsername,
						Company:  RegisteredCompany,
						Role:     UserRole,
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
					SensorData: map[string]any{
						RegisteredQueryField: 23,
					},
				},
				{
					DeviceID:  RegisteredDeviceId,
					Timestamp: AlsoInsideTimeRange,
					SensorData: map[string]any{
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
					SensorData: map[string]any{
						RegisteredQueryField:        float64(23),
						AnotherRegisteredQueryField: float64(80),
					},
				},
				{
					DeviceID:  RegisteredDeviceId,
					Timestamp: AlsoInsideTimeRange,
					SensorData: map[string]any{
						RegisteredQueryField:        float64(25),
						AnotherRegisteredQueryField: float64(70),
					},
				},
			},
			mockAuthStore: authstore.Schema{
				UserInfo: []authstore.UserInfo{
					{
						Username: RegisteredUsername,
						Company:  RegisteredCompany,
						Role:     UserRole,
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
					SensorData: map[string]any{
						RegisteredQueryField:        23,
						AnotherRegisteredQueryField: 80,
					},
				},
				{
					DeviceID:  RegisteredDeviceId,
					Timestamp: AlsoInsideTimeRange,
					SensorData: map[string]any{
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
					SensorData: map[string]any{
						RegisteredQueryField: float64(23),
					},
				},
				{
					DeviceID:  RegisteredDeviceId,
					Timestamp: AlsoInsideTimeRange,
					SensorData: map[string]any{
						RegisteredQueryField: float64(25),
					},
				},
			},
			mockAuthStore: authstore.Schema{
				UserInfo: []authstore.UserInfo{
					{
						Username: RegisteredUsername,
						Company:  RegisteredCompany,
						Role:     UserRole,
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
					SensorData: map[string]any{
						RegisteredQueryField: 22,
					},
				},
				{
					DeviceID:  RegisteredDeviceId,
					Timestamp: InsideTimeRange,
					SensorData: map[string]any{
						RegisteredQueryField: 23,
					},
				},
				{
					DeviceID:  RegisteredDeviceId,
					Timestamp: AlsoInsideTimeRange,
					SensorData: map[string]any{
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
					SensorData: map[string]any{
						RegisteredQueryField: float64(23),
					},
				},
				{
					DeviceID:  RegisteredDeviceId,
					Timestamp: AlsoInsideTimeRange,
					SensorData: map[string]any{
						RegisteredQueryField: float64(25),
					},
				},
			},
			mockAuthStore: authstore.Schema{
				UserInfo: []authstore.UserInfo{
					{
						Username: RegisteredUsername,
						Company:  RegisteredCompany,
						Role:     UserRole,
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
					SensorData: map[string]any{
						RegisteredQueryField: 22,
					},
				},
				{
					DeviceID:  RegisteredDeviceId,
					Timestamp: InsideTimeRange,
					SensorData: map[string]any{
						RegisteredQueryField: 23,
					},
				},
				{
					DeviceID:  RegisteredDeviceId,
					Timestamp: AlsoInsideTimeRange,
					SensorData: map[string]any{
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
						Role:     UserRole,
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
					SensorData: map[string]any{
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
						Role:     UserRole,
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
					SensorData: map[string]any{
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
					SensorData: map[string]any{
						RegisteredQueryField: float64(23),
					},
				},
			},
			mockAuthStore: authstore.Schema{
				UserInfo: []authstore.UserInfo{
					{
						Username: RegisteredUsername,
						Company:  OtherCompanyThanDevice,
						Role:     AdminUserRole,
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
					SensorData: map[string]any{
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
	router := mux.NewRouter().PathPrefix("/api/v1").Subrouter()
	config := huma.DefaultConfig("DataFarm SensorData API", "1.0.0")
	humaApiMux := humamux.New(router, config)
	humaTest := humatest.Wrap(t, humaApiMux)
	localhuma.RegisterHumaOperations(humaTest, a.VerifyToken, a.GetDeviceData,
		a.Login, a.GetQueryFields)
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
			route := "/api/v1/device/data/" + tc.deviceId
			resp := humaTest.Get(route,
				fmt.Sprintf(`Authorization: Bearer %s`, tc.token), tc.deviceRequest)
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
				Body: append(deviceinfo.GeneralQueryFields, RegisteredQueryField),
			},
			mockAuthStore: authstore.Schema{
				UserInfo: []authstore.UserInfo{
					{
						Username: RegisteredUsername,
						Company:  OtherCompanyThanDevice,
						Role:     UserRole,
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
	}

	db, err := miniredis.Run()
	require.Nil(t, err)
	defer db.Close()
	router := mux.NewRouter().PathPrefix("/api/v1").Subrouter()
	config := huma.DefaultConfig("DataFarm SensorData API", "1.0.0")
	humaApiMux := humamux.New(router, config)
	humaTest := humatest.Wrap(t, humaApiMux)
	localhuma.RegisterHumaOperations(humaTest, a.VerifyToken, a.GetDeviceData,
		a.Login, a.GetQueryFields)
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
			route := "/api/v1/device/queryfields/" + tc.deviceId
			resp := humaTest.Get(route,
				fmt.Sprintf(`Authorization: Bearer %s`, tc.token))
			if resp.Code != tc.wantStatus {
				t.Fatalf("wantStatus: %d, response status: %d", tc.wantStatus, resp.Code)
			}
			defer resp.Result().Body.Close()
			if !tc.wantErr {
				var qf []string
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
