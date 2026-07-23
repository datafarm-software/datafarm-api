package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/datafarm-software/datafarm-api/authstore"
	"github.com/datafarm-software/datafarm-api/datafetcher"
	deviceinfo "github.com/datafarm-software/datafarm-api/device-info"
	"github.com/datafarm-software/datafarm-api/redis"
	"github.com/datafarm-software/datafarm-api/tokenprovider"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func DefaultBatchRequest() datafetcher.BatchSensorDataRequest {
	return datafetcher.BatchSensorDataRequest{
		Hardware: []datafetcher.Hardware{
			{
				DeviceId:    RegisteredDeviceId,
				QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField},
			},
			{
				DeviceId:    AnotherRegisteredDeviceId,
				QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField},
			},
		},
		TimeFrame: datafetcher.TimeFrame{
			Start: RelativeStart,
		},
	}
}

func TestBatchGetSensorData(t *testing.T) {
	tests := map[string]struct {
		wantErr         bool
		wantStatus      int
		mockDataFetcher []datafetcher.SensorData
		want            datafetcher.BatchSensorDataResponse
		mockAuthStore   authstore.Schema
		mockDeviceInfo  deviceinfo.Schema
		mockTokens      map[string]bool
		token           string
		deviceRequests  datafetcher.BatchSensorDataRequest
	}{

		"get multiple deviceIds' data": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: datafetcher.BatchSensorDataResponse{
				Errors: []datafetcher.SensorDataError{},
				Results: []datafetcher.SensorData{
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
			token:          ValidToken,
			deviceRequests: DefaultBatchRequest(),
		},

		"admin user can get sensor data from any company": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: datafetcher.BatchSensorDataResponse{
				Errors: []datafetcher.SensorDataError{},
				Results: []datafetcher.SensorData{
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
			token:          ValidToken,
			deviceRequests: DefaultBatchRequest(),
		},

		"admin user can get sensor data from any network": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: datafetcher.BatchSensorDataResponse{
				Errors: []datafetcher.SensorDataError{},
				Results: []datafetcher.SensorData{
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
			token:          ValidToken,
			deviceRequests: DefaultBatchRequest(),
		},

		"network user can get any sensor data from within network": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: datafetcher.BatchSensorDataResponse{
				Errors: []datafetcher.SensorDataError{},
				Results: []datafetcher.SensorData{
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
			token:          ValidToken,
			deviceRequests: DefaultBatchRequest(),
		},

		"network user cant get sensor data from other network": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: datafetcher.BatchSensorDataResponse{
				Errors: []datafetcher.SensorDataError{
					{
						DeviceId: RegisteredDeviceId,
						Error:    "Unauthorized access to this device.",
					},
					{
						DeviceId: AnotherRegisteredDeviceId,
						Error:    "Unauthorized access to this device.",
					},
				},
				Results: []datafetcher.SensorData{},
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
			token:          ValidToken,
			deviceRequests: DefaultBatchRequest(),
		},

		"user cant get sensor data from other company": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: datafetcher.BatchSensorDataResponse{
				Errors: []datafetcher.SensorDataError{
					{
						DeviceId: RegisteredDeviceId,
						Error:    "Unauthorized access to this device.",
					},
					{
						DeviceId: AnotherRegisteredDeviceId,
						Error:    "Unauthorized access to this device.",
					},
				},
				Results: []datafetcher.SensorData{},
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
			token:          ValidToken,
			deviceRequests: DefaultBatchRequest(),
		},

		"one successful request, one error": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: datafetcher.BatchSensorDataResponse{
				Errors: []datafetcher.SensorDataError{
					{
						DeviceId: AnotherRegisteredDeviceId,
						Error:    "Unauthorized access to this device.",
					},
				},
				Results: []datafetcher.SensorData{
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
			token:          ValidToken,
			deviceRequests: DefaultBatchRequest(),
		},

		"unknown token": {
			wantErr:        true,
			wantStatus:     http.StatusUnauthorized,
			token:          InvalidToken,
			want:           datafetcher.BatchSensorDataResponse{},
			deviceRequests: DefaultBatchRequest(),
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
				var dd datafetcher.BatchSensorDataResponse
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

		"network user cant get queryfields from other network": {
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
				if diff := cmp.Diff(tc.want, dd, cmpOpts...); diff != "" {
					t.Fatalf("response mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestBatchGetDataBoundary(t *testing.T) {
	tests := map[string]struct {
		wantErr         bool
		wantStatus      int
		mockDataFetcher []datafetcher.SensorData
		want            datafetcher.BatchDataBoundaryResponse
		mockAuthStore   authstore.Schema
		mockDeviceInfo  deviceinfo.Schema
		mockTokens      map[string]bool
		token           string
		deviceIds       []string
	}{

		"user get multiple deviceid data boundary": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: datafetcher.BatchDataBoundaryResponse{
				Results: []datafetcher.DataBoundary{
					{
						DeviceId: RegisteredDeviceId,
						Start:    InsideTimeRange,
						Stop:     AlsoInsideTimeRange,
					},
					{
						DeviceId: AnotherRegisteredDeviceId,
						Start:    InsideTimeRange,
						Stop:     AlsoInsideTimeRange,
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
				{
					DeviceID:  AnotherRegisteredDeviceId,
					Timestamp: InsideTimeRange,
					SensorData: map[string]float64{
						RegisteredQueryField: 23,
						"batv":               3.4,
					},
				},
				{
					DeviceID:  AnotherRegisteredDeviceId,
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
						QueryFields: []string{RegisteredQueryField},
					},
				},
			},
			mockTokens: map[string]bool{
				ValidToken: true,
			},
			token:     ValidToken,
			deviceIds: []string{RegisteredDeviceId, AnotherRegisteredDeviceId},
		},

		"network user get multiple deviceid data boundary": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: datafetcher.BatchDataBoundaryResponse{
				Results: []datafetcher.DataBoundary{
					{
						DeviceId: RegisteredDeviceId,
						Start:    InsideTimeRange,
						Stop:     AlsoInsideTimeRange,
					},
					{
						DeviceId: AnotherRegisteredDeviceId,
						Start:    InsideTimeRange,
						Stop:     AlsoInsideTimeRange,
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
				{
					DeviceID:  AnotherRegisteredDeviceId,
					Timestamp: InsideTimeRange,
					SensorData: map[string]float64{
						RegisteredQueryField: 23,
						"batv":               3.4,
					},
				},
				{
					DeviceID:  AnotherRegisteredDeviceId,
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
						QueryFields: []string{RegisteredQueryField},
					},
				},
			},
			mockTokens: map[string]bool{
				ValidToken: true,
			},
			token:     ValidToken,
			deviceIds: []string{RegisteredDeviceId, AnotherRegisteredDeviceId},
		},

		"admin user can get device queryfields from any network and company": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: datafetcher.BatchDataBoundaryResponse{
				Results: []datafetcher.DataBoundary{
					{
						DeviceId: RegisteredDeviceId,
						Start:    InsideTimeRange,
						Stop:     AlsoInsideTimeRange,
					},
					{
						DeviceId: AnotherRegisteredDeviceId,
						Start:    InsideTimeRange,
						Stop:     AlsoInsideTimeRange,
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
				{
					DeviceID:  AnotherRegisteredDeviceId,
					Timestamp: InsideTimeRange,
					SensorData: map[string]float64{
						RegisteredQueryField: 23,
						"batv":               3.4,
					},
				},
				{
					DeviceID:  AnotherRegisteredDeviceId,
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
						QueryFields: []string{RegisteredQueryField},
					},
				},
			},
			mockTokens: map[string]bool{
				ValidToken: true,
			},
			token:     ValidToken,
			deviceIds: []string{RegisteredDeviceId, AnotherRegisteredDeviceId},
		},

		"network user can get any device databoundary from within network": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: datafetcher.BatchDataBoundaryResponse{
				Results: []datafetcher.DataBoundary{
					{
						DeviceId: RegisteredDeviceId,
						Start:    InsideTimeRange,
						Stop:     AlsoInsideTimeRange,
					},
					{
						DeviceId: AnotherRegisteredDeviceId,
						Start:    InsideTimeRange,
						Stop:     AlsoInsideTimeRange,
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
				{
					DeviceID:  AnotherRegisteredDeviceId,
					Timestamp: InsideTimeRange,
					SensorData: map[string]float64{
						RegisteredQueryField: 23,
						"batv":               3.4,
					},
				},
				{
					DeviceID:  AnotherRegisteredDeviceId,
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
						QueryFields: []string{RegisteredQueryField},
					},
				},
			},
			mockTokens: map[string]bool{
				ValidToken: true,
			},
			token:     ValidToken,
			deviceIds: []string{RegisteredDeviceId, AnotherRegisteredDeviceId},
		},

		"network user cant get databoundary from other network": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: datafetcher.BatchDataBoundaryResponse{
				Errors: []datafetcher.DataBoundaryError{
					{
						DeviceId: RegisteredDeviceId,
						Error:    "Unauthorized access to this device.",
					},
					{
						DeviceId: AnotherRegisteredDeviceId,
						Error:    "Unauthorized access to this device.",
					},
				},
			},
			mockAuthStore: authstore.Schema{
				UserInfo: []authstore.UserInfo{
					{
						Username: RegisteredUsername,
						Company:  RegisteredCompany,
						Role:     int(authstore.NetworkUser),
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
				{
					DeviceID:  AnotherRegisteredDeviceId,
					Timestamp: InsideTimeRange,
					SensorData: map[string]float64{
						RegisteredQueryField: 23,
						"batv":               3.4,
					},
				},
				{
					DeviceID:  AnotherRegisteredDeviceId,
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
						QueryFields: []string{RegisteredQueryField},
					},
				},
			},
			mockTokens: map[string]bool{
				ValidToken: true,
			},
			token:     ValidToken,
			deviceIds: []string{RegisteredDeviceId, AnotherRegisteredDeviceId},
		},

		"user cant get device databoundary from other company": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: datafetcher.BatchDataBoundaryResponse{
				Errors: []datafetcher.DataBoundaryError{
					{
						DeviceId: RegisteredDeviceId,
						Error:    "Unauthorized access to this device.",
					},
					{
						DeviceId: AnotherRegisteredDeviceId,
						Error:    "Unauthorized access to this device.",
					},
				},
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
				{
					DeviceID:  AnotherRegisteredDeviceId,
					Timestamp: InsideTimeRange,
					SensorData: map[string]float64{
						RegisteredQueryField: 23,
						"batv":               3.4,
					},
				},
				{
					DeviceID:  AnotherRegisteredDeviceId,
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
						QueryFields: []string{RegisteredQueryField},
					},
				},
			},
			mockTokens: map[string]bool{
				ValidToken: true,
			},
			token:     ValidToken,
			deviceIds: []string{RegisteredDeviceId, AnotherRegisteredDeviceId},
		},

		"one accessible deviceid, one inaccessible deviceid": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: datafetcher.BatchDataBoundaryResponse{
				Results: []datafetcher.DataBoundary{
					{
						DeviceId: RegisteredDeviceId,
						Start:    InsideTimeRange,
						Stop:     AlsoInsideTimeRange,
					},
				},
				Errors: []datafetcher.DataBoundaryError{
					{
						DeviceId: AnotherRegisteredDeviceId,
						Error:    "Unauthorized access to this device.",
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
				{
					DeviceID:  AnotherRegisteredDeviceId,
					Timestamp: InsideTimeRange,
					SensorData: map[string]float64{
						RegisteredQueryField: 23,
						"batv":               3.4,
					},
				},
				{
					DeviceID:  AnotherRegisteredDeviceId,
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
						QueryFields: []string{RegisteredQueryField},
					},
				},
			},
			mockTokens: map[string]bool{
				ValidToken: true,
			},
			token:     ValidToken,
			deviceIds: []string{RegisteredDeviceId, AnotherRegisteredDeviceId},
		},

		"unknown token": {
			wantErr:    true,
			wantStatus: http.StatusUnauthorized,
			token:      InvalidToken,
			want:       datafetcher.BatchDataBoundaryResponse{},
			deviceIds:  []string{RegisteredDeviceId},
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
			route := "/batch/device/databoundary"
			resp := humaTest.Post(route,
				fmt.Sprintf(`Authorization: Bearer %s`, tc.token), tc.deviceIds)
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

func TestBatchCsvGetSensorData(t *testing.T) {
	tests := map[string]struct {
		want string
		gsdt GetSensorDataTest
	}{

		"multiple deviceid, single queryfield": {
			want: fmt.Sprintf(",%s\n%s\n%s,%s\n%s\n%s,%s\n",
				RegisteredQueryField, RegisteredDeviceId,
				InsideTimeRange.Format(time.RFC3339), "23.000",
				AnotherRegisteredDeviceId, InsideTimeRange.Format(time.RFC3339),
				"25.000"),
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
						DeviceID:  AnotherRegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField: 25,
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
							QueryFields: []string{RegisteredQueryField},
						},
						{
							DeviceId:    AnotherRegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
				token: ValidToken,
				batchRequests: datafetcher.BatchSensorDataRequest{
					Hardware: []datafetcher.Hardware{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
						{
							DeviceId:    AnotherRegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
					},
					TimeFrame: datafetcher.TimeFrame{
						Start: RelativeStart,
					},
				},
			},
		},

		"multiple deviceid, single queryfield in specific timezone": {
			want: fmt.Sprintf(",%s\n%s\n%s,%s\n%s\n%s,%s\n",
				RegisteredQueryField, RegisteredDeviceId,
				InsideTimeRange.Local().Format(time.RFC3339), "23.000",
				AnotherRegisteredDeviceId, InsideTimeRange.Local().Format(time.RFC3339),
				"25.000"),
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
						DeviceID:  AnotherRegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField: 25,
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
							QueryFields: []string{RegisteredQueryField},
						},
						{
							DeviceId:    AnotherRegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
				token: ValidToken,
				batchRequests: datafetcher.BatchSensorDataRequest{
					Hardware: []datafetcher.Hardware{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
						{
							DeviceId:    AnotherRegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
					},
					TimeFrame: datafetcher.TimeFrame{
						Start:    RelativeStart,
						Timezone: "Africa/Johannesburg",
					},
				},
			},
		},

		"multiple deviceid, multiple queryfield": {
			want: fmt.Sprintf(",%s,%s\n%s\n%s,%s,%s\n%s\n%s,%s,%s\n",
				AnotherRegisteredQueryField, RegisteredQueryField, RegisteredDeviceId,
				InsideTimeRange.Format(time.RFC3339), "80.000", "23.000",
				AnotherRegisteredDeviceId, InsideTimeRange.Format(time.RFC3339),
				"81.000", "25.000"),
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
						DeviceID:  AnotherRegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField:        25,
							AnotherRegisteredQueryField: 81,
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
							QueryFields: []string{RegisteredQueryField},
						},
						{
							DeviceId:    AnotherRegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
				token: ValidToken,
				batchRequests: datafetcher.BatchSensorDataRequest{
					Hardware: []datafetcher.Hardware{
						{
							DeviceId: RegisteredDeviceId,
							QueryFields: []string{
								RegisteredQueryField, AnotherRegisteredQueryField},
						},
						{
							DeviceId: AnotherRegisteredDeviceId,
							QueryFields: []string{
								RegisteredQueryField, AnotherRegisteredQueryField},
						},
					},
					TimeFrame: datafetcher.TimeFrame{
						Start: RelativeStart,
					},
				},
			},
		},

		"multiple deviceid, multiple queryfield seperate timestamp": {
			want: fmt.Sprintf(",%s,%s\n%s\n%s,,%s\n%s\n%s,%s,\n",
				AnotherRegisteredQueryField, RegisteredQueryField, RegisteredDeviceId,
				InsideTimeRange.Format(time.RFC3339), "23.000",
				AnotherRegisteredDeviceId, AlsoInsideTimeRange.Format(time.RFC3339),
				"81.000"),
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
						DeviceID:  AnotherRegisteredDeviceId,
						Timestamp: AlsoInsideTimeRange,
						SensorData: map[string]float64{
							AnotherRegisteredQueryField: 81,
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
							QueryFields: []string{RegisteredQueryField},
						},
						{
							DeviceId:    AnotherRegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
				token: ValidToken,
				batchRequests: datafetcher.BatchSensorDataRequest{
					Hardware: []datafetcher.Hardware{
						{
							DeviceId: RegisteredDeviceId,
							QueryFields: []string{
								RegisteredQueryField, AnotherRegisteredQueryField},
						},
						{
							DeviceId: AnotherRegisteredDeviceId,
							QueryFields: []string{
								RegisteredQueryField, AnotherRegisteredQueryField},
						},
					},
					TimeFrame: datafetcher.TimeFrame{
						Start: RelativeStart,
					},
				},
			},
		},

		"multiple deviceid, all errors": {
			want: fmt.Sprintf("%s,%s\n%s,%s\n", RegisteredDeviceId,
				"Unauthorized access to this device.", AnotherRegisteredDeviceId, "Unauthorized access to this device."),
			gsdt: GetSensorDataTest{wantErr: false,
				wantStatus: http.StatusOK,
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
				mockDataFetcher: []datafetcher.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField: 23,
						},
					},
					{
						DeviceID:  AnotherRegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField: 25,
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
							QueryFields: []string{RegisteredQueryField},
						},
						{
							DeviceId:    AnotherRegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
				token: ValidToken,
				batchRequests: datafetcher.BatchSensorDataRequest{
					Hardware: []datafetcher.Hardware{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
						{
							DeviceId:    AnotherRegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
					},
					TimeFrame: datafetcher.TimeFrame{
						Start: RelativeStart,
					},
				},
			},
		},

		"multiple deviceid, single queryfield, result and errors mixed together": {
			want: fmt.Sprintf(",%s\n%s\n%s,%s\n%s,%s\n",
				RegisteredQueryField, RegisteredDeviceId,
				InsideTimeRange.Format(time.RFC3339), "23.000",
				AnotherRegisteredDeviceId, "Unauthorized access to this device."),
			gsdt: GetSensorDataTest{wantErr: false,
				wantStatus: http.StatusOK,
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
				mockDataFetcher: []datafetcher.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField: 23,
						},
					},
					{
						DeviceID:  AnotherRegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField: 25,
						},
					},
				},
				mockDeviceInfo: deviceinfo.Schema{
					DeviceCompanies: []deviceinfo.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: AnotherRegisteredCompany},
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
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
				token: ValidToken,
				batchRequests: datafetcher.BatchSensorDataRequest{
					Hardware: []datafetcher.Hardware{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
						{
							DeviceId:    AnotherRegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
					},
					TimeFrame: datafetcher.TimeFrame{
						Start: RelativeStart,
					},
				},
			},
		},

		"multiple deviceid, multiple queryfield, result and errors mixed together": {
			want: fmt.Sprintf(",%s,%s\n%s\n%s,%s,%s\n%s,%s\n",
				AnotherRegisteredQueryField, RegisteredQueryField, RegisteredDeviceId,
				InsideTimeRange.Format(time.RFC3339), "80.000", "23.000",
				AnotherRegisteredDeviceId, "Unauthorized access to this device."),
			gsdt: GetSensorDataTest{wantErr: false,
				wantStatus: http.StatusOK,
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
						DeviceID:  AnotherRegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField: 25,
						},
					},
				},
				mockDeviceInfo: deviceinfo.Schema{
					DeviceCompanies: []deviceinfo.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: AnotherRegisteredCompany},
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
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
				token: ValidToken,
				batchRequests: datafetcher.BatchSensorDataRequest{
					Hardware: []datafetcher.Hardware{
						{
							DeviceId: RegisteredDeviceId,
							QueryFields: []string{
								RegisteredQueryField, AnotherRegisteredQueryField},
						},
						{DeviceId: AnotherRegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
					},
					TimeFrame: datafetcher.TimeFrame{
						Start: RelativeStart,
					},
				},
			},
		},

		"multiple deviceid, multiple queryfield, seperate timestamp, result and errors mixed together": {
			want: fmt.Sprintf(",%s,%s\n%s\n%s,%s,\n%s,,%s\n%s,%s\n",
				AnotherRegisteredQueryField, RegisteredQueryField, RegisteredDeviceId,
				InsideTimeRange.Format(time.RFC3339), "80.000",
				AlsoInsideTimeRange.Format(time.RFC3339), "23.000",
				AnotherRegisteredDeviceId, "Unauthorized access to this device."),
			gsdt: GetSensorDataTest{wantErr: false,
				wantStatus: http.StatusOK,
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
							RegisteredQueryField: 23,
						},
					},
				},
				mockDeviceInfo: deviceinfo.Schema{
					DeviceCompanies: []deviceinfo.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: AnotherRegisteredCompany},
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
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
				token: ValidToken,
				batchRequests: datafetcher.BatchSensorDataRequest{
					Hardware: []datafetcher.Hardware{
						{
							DeviceId: RegisteredDeviceId,
							QueryFields: []string{
								RegisteredQueryField, AnotherRegisteredQueryField},
						},
						{
							DeviceId:    AnotherRegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
					},
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
	var close CloseFunc
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			close = setupGetSensorDataTest(t, tc.gsdt, db)
			defer close()
			route := "/batch/device/sensordata"
			resp := humaTest.Post(route,
				fmt.Sprintf(`Authorization: Bearer %s`, tc.gsdt.token),
				`Accept: text/csv`,
				tc.gsdt.batchRequests,
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
