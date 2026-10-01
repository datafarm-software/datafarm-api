package tests

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/datafarm-software/datafarm-api/api"
	"github.com/datafarm-software/datafarm-api/api/authstore"
	"github.com/datafarm-software/datafarm-api/api/device"
	"github.com/datafarm-software/datafarm-api/api/device/data"
	"github.com/datafarm-software/datafarm-api/api/device/info"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func TestLogin(t *testing.T) {
	tests := map[string]struct {
		MockApi
		wantToken          string
		username, password string
		wantStatus         int
		wantErr            bool
	}{

		"successfully login": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			wantToken:  ValidToken,
			username:   RegisteredUsername,
			password:   RegisteredPassword,
			MockApi: MockApi{
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
		},

		"deny access": {
			wantErr:    true,
			wantStatus: http.StatusUnauthorized,
			username:   UnregisteredUsername,
			password:   UnregisteredPassword,
			MockApi: MockApi{
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
		},
	}

	var humaTest humatest.TestAPI
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			a, close := tc.MockApi.Setup(t)
			defer close()
			humaTest = setupHuma(t, a)
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
		MockApi
		GetSensorDataTest
		want []data.SensorData
	}{

		"successfully get deviceid data": {
			want: []data.SensorData{
				{
					DeviceID:  RegisteredDeviceId,
					Timestamp: InsideTimeRange,
					SensorData: map[string]float64{
						RegisteredQueryField: 23,
					},
				},
			},
			MockApi: MockApi{
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
				mockDataFetcher: []data.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField: 23,
						},
					},
				},
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			GetSensorDataTest: GetSensorDataTest{
				wantErr:    false,
				wantStatus: http.StatusOK,
				token:      ValidToken,
				deviceId:   RegisteredDeviceId,
				SensorDataRequest: &data.SensorDataRequest{
					Hardware: data.Hardware{
						QueryFields: []string{RegisteredQueryField},
					},
					TimeFrame: data.TimeFrame{
						Start: RelativeStart,
					},
				},
			},
		},

		"successfully get deviceid data in Africa/Johannesburg timezone": {
			want: []data.SensorData{
				{
					DeviceID:  RegisteredDeviceId,
					Timestamp: InsideTimeRange.Local(),
					SensorData: map[string]float64{
						RegisteredQueryField: 23,
					},
				},
			},
			MockApi: MockApi{
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
				mockDataFetcher: []data.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField: 23,
						},
					},
				},
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			GetSensorDataTest: GetSensorDataTest{
				wantErr:    false,
				wantStatus: http.StatusOK,
				token:      ValidToken,
				deviceId:   RegisteredDeviceId,
				SensorDataRequest: &data.SensorDataRequest{
					Hardware: data.Hardware{QueryFields: []string{RegisteredQueryField}},
					TimeFrame: data.TimeFrame{
						Start:    RelativeStart,
						Timezone: data.Timezone{Timezone: "Africa/Johannesburg"},
					},
				},
			},
		},

		"invalid timezone requested so unprocessable": {
			want: nil,
			MockApi: MockApi{
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
				mockDataFetcher: []data.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField: 23,
						},
					},
				},
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			GetSensorDataTest: GetSensorDataTest{
				wantErr:    true,
				wantStatus: http.StatusUnprocessableEntity,
				token:      ValidToken,
				deviceId:   RegisteredDeviceId,
				SensorDataRequest: &data.SensorDataRequest{
					Hardware: data.Hardware{QueryFields: []string{RegisteredQueryField}},
					TimeFrame: data.TimeFrame{
						Start:    RelativeStart,
						Timezone: data.Timezone{Timezone: InvalidTimezone},
					},
				},
			},
		},

		"unprocessable because more than 20 queryFields requested": {
			want: nil,
			MockApi: MockApi{
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
				mockDataFetcher: []data.SensorData{
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
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
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
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			GetSensorDataTest: GetSensorDataTest{
				wantErr:    true,
				wantStatus: http.StatusUnprocessableEntity,
				token:      ValidToken,
				deviceId:   RegisteredDeviceId,
				SensorDataRequest: &data.SensorDataRequest{
					Hardware: data.Hardware{
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
					TimeFrame: data.TimeFrame{
						Start: RelativeStart,
					},
				},
			},
		},

		"admin user can get all device queryfields": {
			want: []data.SensorData{
				{
					DeviceID:  RegisteredDeviceId,
					Timestamp: InsideTimeRange,
					SensorData: map[string]float64{
						RegisteredQueryField:        23,
						AnotherRegisteredQueryField: 80,
					},
				},
			},
			MockApi: MockApi{
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
				mockDataFetcher: []data.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField:        23,
							AnotherRegisteredQueryField: 80,
						},
					},
				},
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
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
			},
			GetSensorDataTest: GetSensorDataTest{
				wantErr:    false,
				wantStatus: http.StatusOK,
				token:      ValidToken,
				deviceId:   RegisteredDeviceId,
				SensorDataRequest: &data.SensorDataRequest{
					Hardware: data.Hardware{QueryFields: []string{"all"}},
					TimeFrame: data.TimeFrame{
						Start: RelativeStart,
					},
				},
			},
		},

		"network user can get all device queryfields": {
			want: []data.SensorData{
				{
					DeviceID:  RegisteredDeviceId,
					Timestamp: InsideTimeRange,
					SensorData: map[string]float64{
						RegisteredQueryField:        23,
						AnotherRegisteredQueryField: 80,
					},
				},
			},
			MockApi: MockApi{
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
				mockDataFetcher: []data.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField:        23,
							AnotherRegisteredQueryField: 80,
						},
					},
				},
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
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
			},
			GetSensorDataTest: GetSensorDataTest{
				wantErr:    false,
				wantStatus: http.StatusOK,
				token:      ValidToken,
				deviceId:   RegisteredDeviceId,
				SensorDataRequest: &data.SensorDataRequest{
					Hardware: data.Hardware{QueryFields: []string{"all"}},
					TimeFrame: data.TimeFrame{
						Start: RelativeStart,
					},
				},
			},
		},

		"user can get all device queryfields": {
			want: []data.SensorData{
				{
					DeviceID:  RegisteredDeviceId,
					Timestamp: InsideTimeRange,
					SensorData: map[string]float64{
						RegisteredQueryField:        23,
						AnotherRegisteredQueryField: 80,
					},
				},
			},
			MockApi: MockApi{
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
				mockDataFetcher: []data.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField:        23,
							AnotherRegisteredQueryField: 80,
						},
					},
				},
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
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
			},
			GetSensorDataTest: GetSensorDataTest{
				wantErr:    false,
				wantStatus: http.StatusOK,
				token:      ValidToken,
				deviceId:   RegisteredDeviceId,
				SensorDataRequest: &data.SensorDataRequest{
					Hardware: data.Hardware{QueryFields: []string{"all"}},
					TimeFrame: data.TimeFrame{
						Start: RelativeStart,
					},
				},
			},
		},

		"unknown token": {
			want: nil,
			GetSensorDataTest: GetSensorDataTest{
				wantErr:    true,
				wantStatus: http.StatusUnauthorized,
				token:      InvalidToken,
				deviceId:   RegisteredDeviceId,
				SensorDataRequest: &data.SensorDataRequest{
					Hardware: data.Hardware{QueryFields: []string{RegisteredQueryField}},
					TimeFrame: data.TimeFrame{
						Start: RelativeStart,
					},
				},
			},
		},

		"start time in future": {
			want: nil,
			GetSensorDataTest: GetSensorDataTest{
				wantErr:    true,
				wantStatus: http.StatusBadRequest,
				token:      ValidToken,
				deviceId:   RegisteredDeviceId,
				SensorDataRequest: &data.SensorDataRequest{
					Hardware: data.Hardware{QueryFields: []string{RegisteredQueryField}},
					TimeFrame: data.TimeFrame{
						Start: FutureStart,
						Stop:  Stop,
					},
				},
			},
			MockApi: MockApi{
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
			GetSensorDataTest: GetSensorDataTest{
				wantErr:    true,
				wantStatus: http.StatusBadRequest,
				token:      ValidToken,
				deviceId:   RegisteredDeviceId,
				SensorDataRequest: &data.SensorDataRequest{
					Hardware: data.Hardware{QueryFields: []string{RegisteredQueryField}},
					TimeFrame: data.TimeFrame{
						Start: StartGreaterThanStop,
						Stop:  Stop,
					},
				},
			},
			MockApi: MockApi{
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
			want: []data.SensorData{
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
			MockApi: MockApi{
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
				mockDataFetcher: []data.SensorData{
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
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			GetSensorDataTest: GetSensorDataTest{
				wantErr:    false,
				wantStatus: http.StatusOK,
				token:      ValidToken,
				deviceId:   RegisteredDeviceId,
				SensorDataRequest: &data.SensorDataRequest{
					Hardware: data.Hardware{QueryFields: []string{RegisteredQueryField}},
					TimeFrame: data.TimeFrame{
						Start: Start,
						Stop:  StopInFuture,
					},
				},
			},
		},

		"relative start time more than 90 days in the past": {
			want: nil,
			MockApi: MockApi{
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
			GetSensorDataTest: GetSensorDataTest{
				wantErr:    true,
				wantStatus: http.StatusBadRequest,
				token:      ValidToken,
				deviceId:   RegisteredDeviceId,
				SensorDataRequest: &data.SensorDataRequest{
					Hardware: data.Hardware{QueryFields: []string{RegisteredQueryField}},
					TimeFrame: data.TimeFrame{
						Start: RelativeMoreThanNinetyDays,
					},
				},
			},
		},

		"start time more than 90 days in the past": {
			want: nil,
			MockApi: MockApi{
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
			GetSensorDataTest: GetSensorDataTest{
				wantErr:    true,
				wantStatus: http.StatusBadRequest,
				token:      ValidToken,
				deviceId:   RegisteredDeviceId,
				SensorDataRequest: &data.SensorDataRequest{
					Hardware: data.Hardware{QueryFields: []string{RegisteredQueryField}},
					TimeFrame: data.TimeFrame{
						Start: MoreThanNinetyDays,
						Stop:  Stop,
					},
				},
			},
		},

		"get multiple data points within time range": {
			want: []data.SensorData{
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
			MockApi: MockApi{
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
				mockDataFetcher: []data.SensorData{
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
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			GetSensorDataTest: GetSensorDataTest{
				wantErr:    false,
				wantStatus: http.StatusOK,
				token:      ValidToken,
				deviceId:   RegisteredDeviceId,
				SensorDataRequest: &data.SensorDataRequest{
					Hardware: data.Hardware{QueryFields: []string{RegisteredQueryField}},
					TimeFrame: data.TimeFrame{
						Start: RelativeStart,
					},
				},
			},
		},

		"get multiple queryfields' data": {
			want: []data.SensorData{
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
			MockApi: MockApi{
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
				mockDataFetcher: []data.SensorData{
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
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			GetSensorDataTest: GetSensorDataTest{
				wantErr:    false,
				wantStatus: http.StatusOK,
				token:      ValidToken,
				deviceId:   RegisteredDeviceId,
				SensorDataRequest: &data.SensorDataRequest{
					Hardware: data.Hardware{
						QueryFields: []string{
							RegisteredQueryField, AnotherRegisteredQueryField}},
					TimeFrame: data.TimeFrame{
						Start: RelativeStart,
					},
				},
			},
		},

		"exclude data points outside requested time range": {
			want: []data.SensorData{
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
			MockApi: MockApi{
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
				mockDataFetcher: []data.SensorData{
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
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			GetSensorDataTest: GetSensorDataTest{
				wantErr:    false,
				wantStatus: http.StatusOK,
				token:      ValidToken,
				deviceId:   RegisteredDeviceId,
				SensorDataRequest: &data.SensorDataRequest{
					Hardware: data.Hardware{QueryFields: []string{RegisteredQueryField}},
					TimeFrame: data.TimeFrame{
						Start: RelativeStart,
					},
				},
			},
		},

		"exclude data points outside requested time range, using relative start time": {
			want: []data.SensorData{
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
			MockApi: MockApi{
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
				mockDataFetcher: []data.SensorData{
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
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			GetSensorDataTest: GetSensorDataTest{
				wantErr:    false,
				wantStatus: http.StatusOK,
				token:      ValidToken,
				deviceId:   RegisteredDeviceId,
				SensorDataRequest: &data.SensorDataRequest{
					Hardware: data.Hardware{QueryFields: []string{RegisteredQueryField}},
					TimeFrame: data.TimeFrame{
						Start: RelativeStart,
					},
				},
			},
		},

		"no data in requested range": {
			want: nil,
			MockApi: MockApi{
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
				mockDataFetcher: []data.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField: 23,
						},
					},
				},
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			GetSensorDataTest: GetSensorDataTest{
				wantErr:    true,
				wantStatus: http.StatusNoContent,
				token:      ValidToken,
				deviceId:   RegisteredDeviceId,
				SensorDataRequest: &data.SensorDataRequest{
					Hardware: data.Hardware{QueryFields: []string{RegisteredQueryField}},
					TimeFrame: data.TimeFrame{
						Start: "-1h",
					},
				},
			},
		},

		"device doesnt exist": {
			want: nil,
			MockApi: MockApi{
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
				mockDataFetcher: []data.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField: 23,
						},
					},
				},
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			GetSensorDataTest: GetSensorDataTest{
				wantErr:    true,
				wantStatus: http.StatusNotFound,
				token:      ValidToken,
				deviceId:   UnregisteredDeviceId,
				SensorDataRequest: &data.SensorDataRequest{
					Hardware: data.Hardware{QueryFields: []string{RegisteredQueryField}},
					TimeFrame: data.TimeFrame{
						Start: "-1h",
					},
				},
			},
		},

		"non admin can't request deviceid not in user company": {
			want: nil,
			MockApi: MockApi{
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
				mockDataFetcher: []data.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField: 23,
						},
					},
				},
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			GetSensorDataTest: GetSensorDataTest{
				wantErr:    true,
				wantStatus: http.StatusUnauthorized,
				token:      ValidToken,
				deviceId:   RegisteredDeviceId,
				SensorDataRequest: &data.SensorDataRequest{
					Hardware: data.Hardware{QueryFields: []string{RegisteredQueryField}},
					TimeFrame: data.TimeFrame{
						Start: RelativeStart,
					},
				},
			},
		},

		"admin user can request deviceid not in user company": {
			want: []data.SensorData{
				{
					DeviceID:  RegisteredDeviceId,
					Timestamp: InsideTimeRange,
					SensorData: map[string]float64{
						RegisteredQueryField: 23,
					},
				},
			},
			MockApi: MockApi{
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
				mockDataFetcher: []data.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField: 23,
						},
					},
				},
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			GetSensorDataTest: GetSensorDataTest{
				wantErr:    false,
				wantStatus: http.StatusOK,
				token:      ValidToken,
				deviceId:   RegisteredDeviceId,
				SensorDataRequest: &data.SensorDataRequest{
					Hardware: data.Hardware{QueryFields: []string{RegisteredQueryField}},
					TimeFrame: data.TimeFrame{
						Start: RelativeStart,
					},
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			api, closeFunc := tc.MockApi.Setup(t)
			defer closeFunc()
			humaTest := setupHuma(t, api)
			qp := makeQueryParams(any(tc.SensorDataRequest), t)
			route := "/device/" + tc.deviceId + "/sensordata" + qp
			resp := humaTest.Get(route,
				fmt.Sprintf(`Authorization: Bearer %s`, tc.token))
			if resp.Code != tc.wantStatus {
				t.Fatalf("wantStatus: %d, response status: %d", tc.wantStatus, resp.Code)
			}
			defer resp.Result().Body.Close()
			if !tc.wantErr {
				var dd []data.SensorData
				body := resp.Body.Bytes()
				err := json.Unmarshal(body, &dd)
				require.Nil(t, err)
				if diff := cmp.Diff(tc.want, dd, cmpOpts...); diff != "" {
					t.Fatalf("response mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestGetLatestSensorData(t *testing.T) {
	tests := map[string]struct {
		MockApi
		GetSensorDataTest
		want data.SensorData
	}{

		"successfully get last deviceid data": {
			want: data.SensorData{
				DeviceID:  RegisteredDeviceId,
				Timestamp: AlsoInsideTimeRange,
				SensorData: map[string]float64{
					RegisteredQueryField: 23,
				},
			},
			MockApi: MockApi{
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
				mockDataFetcher: []data.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField: 24,
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
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			GetSensorDataTest: GetSensorDataTest{
				wantStatus: http.StatusOK,
				token:      ValidToken,
				deviceId:   RegisteredDeviceId,
				LatestSensorDataRequest: &data.LatestSensorDataRequest{
					Hardware: data.Hardware{
						QueryFields: []string{RegisteredQueryField},
					},
				},
			},
		},

		"successfully get deviceid data in Africa/Johannesburg timezone": {
			want: data.SensorData{
				DeviceID:  RegisteredDeviceId,
				Timestamp: AlsoInsideTimeRange.Local(),
				SensorData: map[string]float64{
					RegisteredQueryField: 23,
				},
			},
			MockApi: MockApi{
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
				mockDataFetcher: []data.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField: 25,
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
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			GetSensorDataTest: GetSensorDataTest{
				wantErr:    false,
				wantStatus: http.StatusOK,
				token:      ValidToken,
				deviceId:   RegisteredDeviceId,
				LatestSensorDataRequest: &data.LatestSensorDataRequest{
					Hardware: data.Hardware{QueryFields: []string{RegisteredQueryField}},
					Timezone: data.Timezone{Timezone: "Africa/Johannesburg"},
				},
			},
		},

		"invalid timezone requested so unprocessable": {
			want: data.SensorData{},
			MockApi: MockApi{
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
				mockDataFetcher: []data.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField: 23,
						},
					},
				},
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			GetSensorDataTest: GetSensorDataTest{
				wantErr:    true,
				wantStatus: http.StatusUnprocessableEntity,
				token:      ValidToken,
				deviceId:   RegisteredDeviceId,
				LatestSensorDataRequest: &data.LatestSensorDataRequest{
					Hardware: data.Hardware{
						QueryFields: []string{RegisteredQueryField}},
					Timezone: data.Timezone{Timezone: InvalidTimezone},
				},
			},
		},

		"unprocessable because more than 20 queryFields requested": {
			want: data.SensorData{},
			MockApi: MockApi{
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
				mockDataFetcher: []data.SensorData{
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
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
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
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			GetSensorDataTest: GetSensorDataTest{
				wantErr:    true,
				wantStatus: http.StatusUnprocessableEntity,
				token:      ValidToken,
				deviceId:   RegisteredDeviceId,
				LatestSensorDataRequest: &data.LatestSensorDataRequest{
					Hardware: data.Hardware{
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
		},

		"admin user can get all device queryfields": {
			want: data.SensorData{
				DeviceID:  RegisteredDeviceId,
				Timestamp: AlsoInsideTimeRange,
				SensorData: map[string]float64{
					RegisteredQueryField:        23,
					AnotherRegisteredQueryField: 80,
				},
			},
			MockApi: MockApi{
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
				mockDataFetcher: []data.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField:        20,
							AnotherRegisteredQueryField: 70,
						},
					},
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: AlsoInsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField:        23,
							AnotherRegisteredQueryField: 80,
						},
					},
				},
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
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
			},
			GetSensorDataTest: GetSensorDataTest{
				wantStatus: http.StatusOK,
				token:      ValidToken,
				deviceId:   RegisteredDeviceId,
				LatestSensorDataRequest: &data.LatestSensorDataRequest{
					Hardware: data.Hardware{QueryFields: []string{"all"}},
				},
			},
		},

		"network user can get all device queryfields": {
			want: data.SensorData{
				DeviceID:  RegisteredDeviceId,
				Timestamp: AlsoInsideTimeRange,
				SensorData: map[string]float64{
					RegisteredQueryField:        23,
					AnotherRegisteredQueryField: 80,
				},
			},
			MockApi: MockApi{
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
				mockDataFetcher: []data.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField:        33,
							AnotherRegisteredQueryField: 90,
						},
					},
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: AlsoInsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField:        23,
							AnotherRegisteredQueryField: 80,
						},
					},
				},
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
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
			},
			GetSensorDataTest: GetSensorDataTest{
				wantStatus: http.StatusOK,
				token:      ValidToken,
				deviceId:   RegisteredDeviceId,
				LatestSensorDataRequest: &data.LatestSensorDataRequest{
					Hardware: data.Hardware{QueryFields: []string{"all"}},
				},
			},
		},

		"user can get all device queryfields": {
			want: data.SensorData{
				DeviceID:  RegisteredDeviceId,
				Timestamp: AlsoInsideTimeRange,
				SensorData: map[string]float64{
					RegisteredQueryField:        23,
					AnotherRegisteredQueryField: 80,
				},
			},
			MockApi: MockApi{
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
				mockDataFetcher: []data.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField:        1,
							AnotherRegisteredQueryField: 2,
						},
					},
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: AlsoInsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField:        23,
							AnotherRegisteredQueryField: 80,
						},
					},
				},
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
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
			},
			GetSensorDataTest: GetSensorDataTest{
				wantStatus: http.StatusOK,
				token:      ValidToken,
				deviceId:   RegisteredDeviceId,
				LatestSensorDataRequest: &data.LatestSensorDataRequest{
					Hardware: data.Hardware{QueryFields: []string{"all"}},
				},
			},
		},

		"unknown token": {
			want: data.SensorData{},
			GetSensorDataTest: GetSensorDataTest{
				wantErr:    true,
				wantStatus: http.StatusUnauthorized,
				token:      InvalidToken,
				deviceId:   RegisteredDeviceId,
				LatestSensorDataRequest: &data.LatestSensorDataRequest{
					Hardware: data.Hardware{QueryFields: []string{RegisteredQueryField}},
				},
			},
		},

		"no data": {
			want: data.SensorData{},
			MockApi: MockApi{
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
				mockDataFetcher: []data.SensorData{},
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			GetSensorDataTest: GetSensorDataTest{
				wantErr:    true,
				wantStatus: http.StatusNoContent,
				token:      ValidToken,
				deviceId:   RegisteredDeviceId,
				LatestSensorDataRequest: &data.LatestSensorDataRequest{
					Hardware: data.Hardware{QueryFields: []string{RegisteredQueryField}},
				},
			},
		},

		"device doesnt exist": {
			want: data.SensorData{},
			MockApi: MockApi{
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
				mockDataFetcher: []data.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField: 23,
						},
					},
				},
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			GetSensorDataTest: GetSensorDataTest{
				wantErr:    true,
				wantStatus: http.StatusNotFound,
				token:      ValidToken,
				deviceId:   UnregisteredDeviceId,
				LatestSensorDataRequest: &data.LatestSensorDataRequest{
					Hardware: data.Hardware{QueryFields: []string{RegisteredQueryField}},
				},
			},
		},

		"non admin can't request deviceid not in user company": {
			want: data.SensorData{},
			MockApi: MockApi{
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
				mockDataFetcher: []data.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField: 23,
						},
					},
				},
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			GetSensorDataTest: GetSensorDataTest{
				wantErr:    true,
				wantStatus: http.StatusUnauthorized,
				token:      ValidToken,
				deviceId:   RegisteredDeviceId,
				LatestSensorDataRequest: &data.LatestSensorDataRequest{
					Hardware: data.Hardware{QueryFields: []string{RegisteredQueryField}},
				},
			},
		},

		"admin user can request deviceid not in user company": {
			want: data.SensorData{
				DeviceID:  RegisteredDeviceId,
				Timestamp: AlsoInsideTimeRange,
				SensorData: map[string]float64{
					RegisteredQueryField: 23,
				},
			},
			MockApi: MockApi{
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
				mockDataFetcher: []data.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField: 20,
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
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			GetSensorDataTest: GetSensorDataTest{
				wantStatus: http.StatusOK,
				token:      ValidToken,
				deviceId:   RegisteredDeviceId,
				LatestSensorDataRequest: &data.LatestSensorDataRequest{
					Hardware: data.Hardware{QueryFields: []string{RegisteredQueryField}},
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			api, closeFunc := tc.MockApi.Setup(t)
			defer closeFunc()
			humaTest := setupHuma(t, api)
			qp := makeQueryParams(any(tc.LatestSensorDataRequest), t)
			route := "/device/" + tc.deviceId + "/sensordata/latest" + qp
			resp := humaTest.Get(route,
				fmt.Sprintf(`Authorization: Bearer %s`, tc.token))
			if resp.Code != tc.wantStatus {
				t.Fatalf("wantStatus: %d, response status: %d", tc.wantStatus, resp.Code)
			}
			defer resp.Result().Body.Close()
			if !tc.wantErr {
				var dd data.SensorData
				body := resp.Body.Bytes()
				err := json.Unmarshal(body, &dd)
				require.Nil(t, err)
				if diff := cmp.Diff(tc.want, dd, cmpOpts...); diff != "" {
					t.Fatalf("response mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestMakeQueryParams(t *testing.T) {
	tests := map[string]struct {
		input any
		want  string
	}{

		"sensordatarequest using relative time": {
			input: &data.SensorDataRequest{
				Hardware: data.Hardware{QueryFields: []string{RegisteredQueryField}},
				TimeFrame: data.TimeFrame{
					Start:    RelativeStart,
					Timezone: data.Timezone{Timezone: ValidTimezone},
				},
			},
			want: fmt.Sprintf(`?queryField=%s&timezone-return=%s&start=%s`,
				RegisteredQueryField, url.QueryEscape(ValidTimezone), RelativeStart),
		},

		"sensordatarequest using multiple queryfields": {
			input: &data.SensorDataRequest{
				Hardware: data.Hardware{QueryFields: []string{
					RegisteredQueryField, AnotherRegisteredQueryField}},
				TimeFrame: data.TimeFrame{
					Start:    RelativeStart,
					Timezone: data.Timezone{Timezone: ValidTimezone},
				},
			},
			want: fmt.Sprintf(`?queryField=%s&queryField=%s&timezone-return=%s&start=%s`,
				RegisteredQueryField, AnotherRegisteredQueryField,
				url.QueryEscape(ValidTimezone), RelativeStart),
		},

		"sensordatarequest using absolute time": {
			input: &data.SensorDataRequest{
				Hardware: data.Hardware{QueryFields: []string{RegisteredQueryField}},
				TimeFrame: data.TimeFrame{
					Start:    Start,
					Stop:     Stop,
					Timezone: data.Timezone{Timezone: ValidTimezone},
				},
			},
			want: fmt.Sprintf(`?queryField=%s&timezone-return=%s&start=%s&stop=%s`,
				RegisteredQueryField, url.QueryEscape(ValidTimezone),
				url.QueryEscape(Start), url.QueryEscape(Stop)),
		},

		"lastsensordatarequest with single queryField": {
			input: &data.LatestSensorDataRequest{
				Hardware: data.Hardware{QueryFields: []string{RegisteredQueryField}},
				Timezone: data.Timezone{Timezone: ValidTimezone},
			},
			want: fmt.Sprintf(`?queryField=%s&timezone-return=%s`,
				RegisteredQueryField, url.QueryEscape(ValidTimezone)),
		},

		"lastsensordatarequest with multiple queryField": {
			input: &data.LatestSensorDataRequest{
				Hardware: data.Hardware{QueryFields: []string{
					RegisteredQueryField, AnotherRegisteredQueryField}},
				Timezone: data.Timezone{Timezone: ValidTimezone},
			},
			want: fmt.Sprintf(`?queryField=%s&queryField=%s&timezone-return=%s`,
				RegisteredQueryField, AnotherRegisteredQueryField,
				url.QueryEscape(ValidTimezone)),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := makeQueryParams(tc.input, t)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("response mismatch (-want +got): %s\n", diff)
			}
		})
	}
}

func TestGetQueryFields(t *testing.T) {
	tests := map[string]struct {
		MockApi
		want       info.QueryFields
		deviceId   string
		token      string
		wantStatus int
		wantErr    bool
	}{

		"successfully get queryfields": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			deviceId:   RegisteredDeviceId,
			want: info.QueryFields{
				DeviceId:    RegisteredDeviceId,
				QueryFields: append(info.GeneralQueryFields, RegisteredQueryField),
			},
			token: ValidToken,
			MockApi: MockApi{
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
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
			},
		},

		"regular user cannot request queryfields for device from other company": {
			wantErr:    true,
			wantStatus: http.StatusUnauthorized,
			deviceId:   RegisteredDeviceId,
			want:       info.QueryFields{},
			token:      ValidToken,
			MockApi: MockApi{
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
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
			},
		},

		"admin user can request queryfields for device from other company": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			deviceId:   RegisteredDeviceId,
			want: info.QueryFields{
				DeviceId:    RegisteredDeviceId,
				QueryFields: append(info.GeneralQueryFields, RegisteredQueryField),
			},
			token: ValidToken,
			MockApi: MockApi{
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
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
			},
		},

		"unknown token": {
			wantErr:    true,
			wantStatus: http.StatusUnauthorized,
			token:      InvalidToken,
			want:       info.QueryFields{},
			deviceId:   RegisteredDeviceId,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			api, closeFunc := tc.MockApi.Setup(t)
			defer closeFunc()
			humaTest := setupHuma(t, api)
			route := "/device/" + tc.deviceId + "/queryfields"
			resp := humaTest.Get(route,
				fmt.Sprintf(`Authorization: Bearer %s`, tc.token))
			if resp.Code != tc.wantStatus {
				t.Fatalf("wantStatus: %d, response status: %d", tc.wantStatus, resp.Code)
			}
			defer resp.Result().Body.Close()
			if !tc.wantErr {
				var qf info.QueryFields
				body := resp.Body.Bytes()
				err := json.Unmarshal(body, &qf)
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
		MockApi
		want       info.DeviceIdsResponse
		token      string
		wantStatus int
		wantErr    bool
	}{

		"user gets deviceIds only in company, in network": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want:       info.DeviceIdsResponse{DeviceIds: []string{RegisteredDeviceId}},
			token:      ValidToken,
			MockApi: MockApi{
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
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
						{DeviceId: AnotherRegisteredDeviceId, Company: AnotherRegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
						{DeviceId: AnotherRegisteredDeviceId, Network: AnotherRegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
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
			},
		},

		"no content if user no access to any deviceid": {
			wantStatus: http.StatusNoContent,
			want:       info.DeviceIdsResponse{DeviceIds: []string{}},
			token:      ValidToken,
			MockApi: MockApi{
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
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
		},

		"network user gets deviceIds in network": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: info.DeviceIdsResponse{DeviceIds: []string{
				RegisteredDeviceId, AnotherRegisteredDeviceId}},
			token: ValidToken,
			MockApi: MockApi{
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
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
						{DeviceId: AnotherRegisteredDeviceId, Company: AnotherRegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
						{DeviceId: AnotherRegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
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
			},
		},

		"admin gets all deviceIds": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: info.DeviceIdsResponse{DeviceIds: []string{
				RegisteredDeviceId, AnotherRegisteredDeviceId}},
			token: ValidToken,
			MockApi: MockApi{
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
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
						{DeviceId: AnotherRegisteredDeviceId, Company: AnotherRegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
						{DeviceId: AnotherRegisteredDeviceId, Network: AnotherRegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
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
			},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			api, closeFunc := tc.MockApi.Setup(t)
			defer closeFunc()
			humaTest := setupHuma(t, api)
			route := "/device/ids"
			resp := humaTest.Get(route,
				fmt.Sprintf(`Authorization: Bearer %s`, tc.token))
			if resp.Code != tc.wantStatus {
				t.Fatalf("wantStatus: %d, response status: %d", tc.wantStatus, resp.Code)
			}
			defer resp.Result().Body.Close()
			if !tc.wantErr && tc.wantStatus != http.StatusNoContent {
				var dr info.DeviceIdsResponse
				body := resp.Body.Bytes()
				err := json.Unmarshal(body, &dr)
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
		MockApi
		want       data.DataBoundary
		req        data.DataBoundaryRequest
		token      string
		wantStatus int
		wantErr    bool
	}{

		"user get deviceid data boundary": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: data.DataBoundary{
				DeviceId: RegisteredDeviceId,
				Start:    InsideTimeRange,
				Stop:     AlsoInsideTimeRange,
			},
			token: ValidToken,
			MockApi: MockApi{
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
				mockDataFetcher: []data.SensorData{
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
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			req: data.DataBoundaryRequest{
				DeviceIdParam: data.DeviceIdParam{DeviceId: RegisteredDeviceId},
			},
		},

		"user get deviceid data boundary in specific timezone": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: data.DataBoundary{
				DeviceId: RegisteredDeviceId,
				Start:    InsideTimeRange.Local(),
				Stop:     AlsoInsideTimeRange.Local(),
			},
			token: ValidToken,
			MockApi: MockApi{
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
				mockDataFetcher: []data.SensorData{
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
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			req: data.DataBoundaryRequest{
				DeviceIdParam: data.DeviceIdParam{DeviceId: RegisteredDeviceId},
				Timezone:      data.Timezone{Timezone: "Africa/Johannesburg"},
			},
		},

		"network user get deviceid data boundary": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: data.DataBoundary{
				DeviceId: RegisteredDeviceId,
				Start:    InsideTimeRange,
				Stop:     AlsoInsideTimeRange,
			},
			token: ValidToken,
			MockApi: MockApi{
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
				mockDataFetcher: []data.SensorData{
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
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			req: data.DataBoundaryRequest{
				DeviceIdParam: data.DeviceIdParam{DeviceId: RegisteredDeviceId},
			},
		},

		"admin user get deviceid data boundary": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: data.DataBoundary{
				DeviceId: RegisteredDeviceId,
				Start:    InsideTimeRange,
				Stop:     AlsoInsideTimeRange,
			},
			token: ValidToken,
			MockApi: MockApi{
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
				mockDataFetcher: []data.SensorData{
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
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			req: data.DataBoundaryRequest{
				DeviceIdParam: data.DeviceIdParam{DeviceId: RegisteredDeviceId},
			},
		},

		"no data": {
			wantStatus: http.StatusNoContent,
			want:       data.DataBoundary{},
			token:      ValidToken,
			MockApi: MockApi{
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
				mockDataFetcher: []data.SensorData{},
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			req: data.DataBoundaryRequest{
				DeviceIdParam: data.DeviceIdParam{DeviceId: RegisteredDeviceId},
			},
		},

		"unknown token": {
			wantErr:    true,
			wantStatus: http.StatusUnauthorized,
			token:      InvalidToken,
			want:       data.DataBoundary{},
			req: data.DataBoundaryRequest{
				DeviceIdParam: data.DeviceIdParam{DeviceId: RegisteredDeviceId},
			},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			api, closeFunc := tc.MockApi.Setup(t)
			defer closeFunc()
			humaTest := setupHuma(t, api)
			route := "/device/" + tc.req.DeviceId + "/databoundary"
			route += fmt.Sprintf(`?timezone-return=%s`, tc.req.Timezone.Timezone)
			resp := humaTest.Get(route,
				fmt.Sprintf(`Authorization: Bearer %s`, tc.token))
			if resp.Code != tc.wantStatus {
				t.Fatalf("wantStatus: %d, response status: %d", tc.wantStatus, resp.Code)
			}
			defer resp.Result().Body.Close()
			if !tc.wantErr && tc.wantStatus != http.StatusNoContent {
				var got data.DataBoundary
				body := resp.Body.Bytes()
				err := json.Unmarshal(body, &got)
				require.Nil(t, err)
				if diff := cmp.Diff(tc.want, got, cmpOpts...); diff != "" {
					t.Fatalf("response mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestCsvGetSensorData(t *testing.T) {
	tests := map[string]struct {
		MockApi
		GetSensorDataTest
		want string
	}{

		"single deviceid, single queryfield": {
			want: fmt.Sprintf(",%s\n%s\n%s,%s\n",
				RegisteredQueryField, RegisteredDeviceId,
				InsideTimeRange.Format(time.RFC3339), "23.000"),
			MockApi: MockApi{
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
				mockDataFetcher: []data.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField: 23,
						},
					},
				},
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			GetSensorDataTest: GetSensorDataTest{wantErr: false,
				wantStatus: http.StatusOK,
				token:      ValidToken,
				deviceId:   RegisteredDeviceId,
				SensorDataRequest: &data.SensorDataRequest{
					Hardware: data.Hardware{QueryFields: []string{RegisteredQueryField}},
					TimeFrame: data.TimeFrame{
						Start: RelativeStart,
					},
				},
			},
		},

		"single deviceid, single queryfield with specific timezone": {
			want: fmt.Sprintf(",%s\n%s\n%s,%s\n",
				RegisteredQueryField, RegisteredDeviceId,
				InsideTimeRange.Local().Format(time.RFC3339), "23.000"),
			MockApi: MockApi{
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
				mockDataFetcher: []data.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField: 23,
						},
					},
				},
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			GetSensorDataTest: GetSensorDataTest{wantErr: false,
				wantStatus: http.StatusOK,
				token:      ValidToken,
				deviceId:   RegisteredDeviceId,
				SensorDataRequest: &data.SensorDataRequest{
					Hardware: data.Hardware{QueryFields: []string{RegisteredQueryField}},
					TimeFrame: data.TimeFrame{
						Start:    RelativeStart,
						Timezone: data.Timezone{Timezone: "Africa/Johannesburg"},
					},
				},
			},
		},

		"single deviceid, multiple queryfield": {
			want: fmt.Sprintf(",%s,%s\n%s\n%s,%s,%s\n",
				AnotherRegisteredQueryField, RegisteredQueryField, RegisteredDeviceId,
				InsideTimeRange.Format(time.RFC3339), "80.000", "23.000"),
			MockApi: MockApi{
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
				mockDataFetcher: []data.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField:        23,
							AnotherRegisteredQueryField: 80,
						},
					},
				},
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			GetSensorDataTest: GetSensorDataTest{wantErr: false,
				wantStatus: http.StatusOK,
				token:      ValidToken,
				deviceId:   RegisteredDeviceId,
				SensorDataRequest: &data.SensorDataRequest{
					Hardware: data.Hardware{QueryFields: []string{
						RegisteredQueryField, AnotherRegisteredQueryField}},
					TimeFrame: data.TimeFrame{
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
			MockApi: MockApi{
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
				mockDataFetcher: []data.SensorData{
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
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			GetSensorDataTest: GetSensorDataTest{wantErr: false,
				wantStatus: http.StatusOK,
				token:      ValidToken,
				deviceId:   RegisteredDeviceId,
				SensorDataRequest: &data.SensorDataRequest{
					Hardware: data.Hardware{QueryFields: []string{
						RegisteredQueryField, AnotherRegisteredQueryField}},
					TimeFrame: data.TimeFrame{
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
			MockApi: MockApi{
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
				mockDataFetcher: []data.SensorData{
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
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			GetSensorDataTest: GetSensorDataTest{wantErr: false,
				wantStatus: http.StatusOK,
				token:      ValidToken,
				deviceId:   RegisteredDeviceId,
				SensorDataRequest: &data.SensorDataRequest{
					Hardware: data.Hardware{QueryFields: []string{
						RegisteredQueryField, AnotherRegisteredQueryField}},
					TimeFrame: data.TimeFrame{
						Start: RelativeStart,
					},
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			api, closeFunc := tc.MockApi.Setup(t)
			defer closeFunc()
			humaTest := setupHuma(t, api)
			qp := makeQueryParams(any(tc.SensorDataRequest), t)
			route := "/device/" + tc.deviceId + "/sensordata" + qp
			resp := humaTest.Get(route,
				fmt.Sprintf(`Authorization: Bearer %s`, tc.token),
				"Accept: text/csv",
			)
			if resp.Code != tc.wantStatus {
				t.Fatalf("wantStatus: %d, response status: %d", tc.wantStatus, resp.Code)
			}
			contentType := resp.Header().Get("Content-Type")
			if contentType != "text/csv" {
				t.Fatalf("response content-type not csv: %s", contentType)
			}
			defer resp.Result().Body.Close()
			if !tc.wantErr {
				body := resp.Body.String()
				if diff := cmp.Diff(tc.want, body, cmpOpts...); diff != "" {
					t.Fatalf("response mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestGetLocation(t *testing.T) {
	tests := map[string]struct {
		MockApi
		GetSensorDataTest
		want data.DeviceLocationResponse
	}{

		"successfully retrieve device location information": {
			MockApi{
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
				mockDataFetcher: []data.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							"latitude":  Latitude,
							"longitude": Longitude,
						},
					},
				},
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			GetSensorDataTest{
				wantStatus: http.StatusOK,
				token:      ValidToken,
				deviceId:   RegisteredDeviceId,
			},
			data.DeviceLocationResponse{
				DeviceId: RegisteredDeviceId,
				Time:     InsideTimeRange,
				Latitude: Latitude, Longitude: Longitude,
			},
		},

		"device no location information so no content": {
			MockApi{
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
				mockDataFetcher: []data.SensorData{
					{
						DeviceID:   RegisteredDeviceId,
						Timestamp:  InsideTimeRange,
						SensorData: map[string]float64{},
					},
				},
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			GetSensorDataTest{
				wantStatus: http.StatusNoContent,
				token:      ValidToken,
				deviceId:   RegisteredDeviceId,
			},
			data.DeviceLocationResponse{},
		},

		"device not found": {
			MockApi{
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
				mockDataFetcher: []data.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							"latitude":  Latitude,
							"longitude": Longitude,
						},
					},
				},
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			GetSensorDataTest{
				wantStatus: http.StatusNotFound,
				token:      ValidToken,
				deviceId:   AnotherRegisteredDeviceId,
			},
			data.DeviceLocationResponse{},
		},

		"device no access": {
			MockApi{
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
				mockDataFetcher: []data.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							"latitude":  Latitude,
							"longitude": Longitude,
						},
					},
				},
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			GetSensorDataTest{
				wantStatus: http.StatusUnauthorized,
				token:      ValidToken,
				deviceId:   RegisteredDeviceId,
			},
			data.DeviceLocationResponse{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			api, closeFunc := tc.MockApi.Setup(t)
			defer closeFunc()
			humaTest := setupHuma(t, api)
			route := "/device/" + tc.deviceId + "/location"
			resp := humaTest.Get(route,
				fmt.Sprintf(`Authorization: Bearer %s`, tc.token),
			)
			if resp.Code != tc.wantStatus {
				t.Fatalf("wantStatus: %d, response status: %d", tc.wantStatus, resp.Code)
			}
			defer resp.Result().Body.Close()
			if !tc.wantErr && tc.wantStatus != http.StatusNoContent {
				var location data.DeviceLocationResponse
				err := json.Unmarshal(resp.Body.Bytes(), &location)
				require.Nil(t, err)
				if diff := cmp.Diff(tc.want, location, cmpOpts...); diff != "" {
					t.Fatalf("response mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
