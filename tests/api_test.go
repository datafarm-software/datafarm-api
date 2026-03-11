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
	"github.com/geraud22/datafarm-api/api"
	"github.com/geraud22/datafarm-api/authstore"
	"github.com/geraud22/datafarm-api/datafetcher"
	deviceinfo "github.com/geraud22/datafarm-api/device-info"
	"github.com/geraud22/datafarm-api/redis"
	"github.com/geraud22/datafarm-api/tokenprovider"
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
const RegisteredSensor = "temp-sensor"
const ValidToken = "someToken0"
const InvalidToken = "invalidToken0"
const RelativeStart = "-6h"

var Start = time.Now().Add(-24 * time.Hour).Format(time.RFC3339)
var Stop = time.Now().Format(time.RFC3339)
var OutsideTimeRange = time.Now().Add(-25 * time.Hour)
var InsideTimeRange = time.Now().Add(-2 * time.Hour)
var AlsoInsideTimeRange = time.Now().Add(-1 * time.Hour)
var a = &api.Api{}

func TestLogin(t *testing.T) {
	tests := map[string]struct {
		wantErr                        bool
		wantStatus                     int
		username, password             string
		mockAuthStore                  authstore.Schema
		mockDeviceInfo, wantDeviceInfo deviceinfo.Schema
	}{

		"successfully login": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			username:   RegisteredUsername,
			password:   RegisteredPassword,
			mockAuthStore: authstore.Schema{
				UserInfo: map[string]authstore.UserInfo{
					RegisteredUsername: {
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
				UserInfo: map[string]authstore.UserInfo{
					RegisteredUsername: {
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
	a.RegisterHumaOperations(humaApi)
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
		mockDataFetcher, want *datafetcher.ConsolidatedDeviceData
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
			want: &datafetcher.ConsolidatedDeviceData{
				DeviceData: []datafetcher.DeviceData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]any{
							RegisteredQueryField: float64(23),
						},
					},
				},
			},
			mockAuthStore: authstore.Schema{
				UserInfo: map[string]authstore.UserInfo{
					RegisteredUsername: {
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
			mockDataFetcher: &datafetcher.ConsolidatedDeviceData{
				DeviceData: []datafetcher.DeviceData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]any{
							RegisteredQueryField: 23,
						},
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
				DeviceToSensors: []deviceinfo.DeviceToSensor{
					{
						DeviceId:        RegisteredDeviceId,
						AttachedSensors: []string{RegisteredSensor},
					},
				},
				SensorToQF: []deviceinfo.SensorToQueryFields{
					{
						Sensor:      RegisteredSensor,
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
				QueryField: RegisteredQueryField,
				Start:      Start,
				Stop:       Stop,
			},
		},

		"unknown token": {
			wantErr:    true,
			wantStatus: http.StatusUnauthorized,
			token:      InvalidToken,
			want:       nil,
			deviceId:   RegisteredDeviceId,
			deviceRequest: datafetcher.DeviceDataRequest{
				QueryField: RegisteredQueryField,
				Start:      Start,
				Stop:       Stop,
			},
		},

		"get multiple data points within time range": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: &datafetcher.ConsolidatedDeviceData{
				DeviceData: []datafetcher.DeviceData{
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
			},
			mockAuthStore: authstore.Schema{
				UserInfo: map[string]authstore.UserInfo{
					RegisteredUsername: {
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
			mockDataFetcher: &datafetcher.ConsolidatedDeviceData{
				DeviceData: []datafetcher.DeviceData{
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
			},
			mockDeviceInfo: deviceinfo.Schema{
				DeviceCompanies: []deviceinfo.DeviceToCompany{
					{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
				},
				DeviceNetworks: []deviceinfo.DeviceToNetwork{
					{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
				},
				DeviceToSensors: []deviceinfo.DeviceToSensor{
					{
						DeviceId:        RegisteredDeviceId,
						AttachedSensors: []string{RegisteredSensor},
					},
				},
				SensorToQF: []deviceinfo.SensorToQueryFields{
					{
						Sensor:      RegisteredSensor,
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
				QueryField: RegisteredQueryField,
				Start:      Start,
				Stop:       Stop,
			},
		},

		"exclude data points outside requested time range": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: &datafetcher.ConsolidatedDeviceData{
				DeviceData: []datafetcher.DeviceData{
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
			},
			mockAuthStore: authstore.Schema{
				UserInfo: map[string]authstore.UserInfo{
					RegisteredUsername: {
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
			mockDataFetcher: &datafetcher.ConsolidatedDeviceData{
				DeviceData: []datafetcher.DeviceData{
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
			},
			mockDeviceInfo: deviceinfo.Schema{
				DeviceCompanies: []deviceinfo.DeviceToCompany{
					{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
				},
				DeviceNetworks: []deviceinfo.DeviceToNetwork{
					{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
				},
				DeviceToSensors: []deviceinfo.DeviceToSensor{
					{
						DeviceId:        RegisteredDeviceId,
						AttachedSensors: []string{RegisteredSensor},
					},
				},
				SensorToQF: []deviceinfo.SensorToQueryFields{
					{
						Sensor:      RegisteredSensor,
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
				QueryField: RegisteredQueryField,
				Start:      Start,
				Stop:       Stop,
			},
		},

		"exclude data points outside requested time range, using relative start time": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: &datafetcher.ConsolidatedDeviceData{
				DeviceData: []datafetcher.DeviceData{
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
			},
			mockAuthStore: authstore.Schema{
				UserInfo: map[string]authstore.UserInfo{
					RegisteredUsername: {
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
			mockDataFetcher: &datafetcher.ConsolidatedDeviceData{
				DeviceData: []datafetcher.DeviceData{
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
			},
			mockDeviceInfo: deviceinfo.Schema{
				DeviceCompanies: []deviceinfo.DeviceToCompany{
					{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
				},
				DeviceNetworks: []deviceinfo.DeviceToNetwork{
					{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
				},
				DeviceToSensors: []deviceinfo.DeviceToSensor{
					{
						DeviceId:        RegisteredDeviceId,
						AttachedSensors: []string{RegisteredSensor},
					},
				},
				SensorToQF: []deviceinfo.SensorToQueryFields{
					{
						Sensor:      RegisteredSensor,
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
				QueryField: RegisteredQueryField,
				Start:      RelativeStart,
			},
		},

		"no data in requested range": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want:       &datafetcher.ConsolidatedDeviceData{},
			mockAuthStore: authstore.Schema{
				UserInfo: map[string]authstore.UserInfo{
					RegisteredUsername: {
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
			mockDataFetcher: &datafetcher.ConsolidatedDeviceData{
				DeviceData: []datafetcher.DeviceData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]any{
							RegisteredQueryField: 23,
						},
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
				DeviceToSensors: []deviceinfo.DeviceToSensor{
					{
						DeviceId:        RegisteredDeviceId,
						AttachedSensors: []string{RegisteredSensor},
					},
				},
				SensorToQF: []deviceinfo.SensorToQueryFields{
					{
						Sensor:      RegisteredSensor,
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
				QueryField: RegisteredQueryField,
				Start:      "-1h",
			},
		},

		"non admin can't request deviceid not in user company": {
			wantErr:    true,
			wantStatus: http.StatusUnauthorized,
			want:       &datafetcher.ConsolidatedDeviceData{},
			mockAuthStore: authstore.Schema{
				UserInfo: map[string]authstore.UserInfo{
					RegisteredUsername: {
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
			mockDataFetcher: &datafetcher.ConsolidatedDeviceData{
				DeviceData: []datafetcher.DeviceData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]any{
							RegisteredQueryField: 23,
						},
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
				DeviceToSensors: []deviceinfo.DeviceToSensor{
					{
						DeviceId:        RegisteredDeviceId,
						AttachedSensors: []string{RegisteredSensor},
					},
				},
				SensorToQF: []deviceinfo.SensorToQueryFields{
					{
						Sensor:      RegisteredSensor,
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
				QueryField: RegisteredQueryField,
				Start:      RelativeStart,
			},
		},

		"admin user can request deviceid not in user company": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: &datafetcher.ConsolidatedDeviceData{
				DeviceData: []datafetcher.DeviceData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]any{
							RegisteredQueryField: float64(23),
						},
					},
				},
			},
			mockAuthStore: authstore.Schema{
				UserInfo: map[string]authstore.UserInfo{
					RegisteredUsername: {
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
			mockDataFetcher: &datafetcher.ConsolidatedDeviceData{
				DeviceData: []datafetcher.DeviceData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]any{
							RegisteredQueryField: 23,
						},
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
				DeviceToSensors: []deviceinfo.DeviceToSensor{
					{
						DeviceId:        RegisteredDeviceId,
						AttachedSensors: []string{RegisteredSensor},
					},
				},
				SensorToQF: []deviceinfo.SensorToQueryFields{
					{
						Sensor:      RegisteredSensor,
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
				QueryField: RegisteredQueryField,
				Start:      RelativeStart,
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
	a.RegisterHumaOperations(humaTest)
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
			route := "/api/v1/device/" + tc.deviceId
			resp := humaTest.Get(route,
				fmt.Sprintf(`Authorization: Bearer %s`, tc.token), tc.deviceRequest)
			if resp.Code != tc.wantStatus {
				t.Fatalf("wantStatus: %d, response status: %d", tc.wantStatus, resp.Code)
			}
			defer resp.Result().Body.Close()
			if !tc.wantErr {
				var cdd datafetcher.ConsolidatedDeviceData
				body := resp.Body.Bytes()
				err = json.Unmarshal(body, &cdd)
				require.Nil(t, err)
				if diff := cmp.Diff(tc.want, &cdd); diff != "" {
					t.Fatalf("response mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
