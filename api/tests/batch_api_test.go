package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/datafarm-software/datafarm-api/api/authstore"
	"github.com/datafarm-software/datafarm-api/api/device"
	"github.com/datafarm-software/datafarm-api/api/device/data"
	"github.com/datafarm-software/datafarm-api/api/device/info"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func DefaultBatchRequest() data.BatchSensorDataRequest {
	return data.BatchSensorDataRequest{
		Hardware: []data.Hardware{
			{
				DeviceIdParam: data.DeviceIdParam{DeviceId: RegisteredDeviceId},
				QueryFields:   []string{RegisteredQueryField, AnotherRegisteredQueryField},
			},
			{
				DeviceIdParam: data.DeviceIdParam{DeviceId: AnotherRegisteredDeviceId},
				QueryFields:   []string{RegisteredQueryField, AnotherRegisteredQueryField},
			},
		},
		TimeFrame: data.TimeFrame{
			Start: RelativeStart,
		},
	}
}

func TestBatchGetSensorData(t *testing.T) {
	tests := map[string]struct {
		MockApi
		deviceRequests          data.BatchSensorDataRequest
		want                    data.BatchSensorDataResponse
		token                   string
		wantStatus              int
		disconnectedDataFetcher bool
		wantErr                 bool
	}{

		"get multiple deviceIds' data": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: data.BatchSensorDataResponse{
				Errors: []data.SensorDataError{},
				Results: []data.SensorData{
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
						DeviceID:  AnotherRegisteredDeviceId,
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
						{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
						{DeviceId: AnotherRegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
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
			},
			token:          ValidToken,
			deviceRequests: DefaultBatchRequest(),
		},

		"get multiple deviceIds' data in different timezone": {
			wantStatus: http.StatusOK,
			want: data.BatchSensorDataResponse{
				Errors: []data.SensorDataError{},
				Results: []data.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange.Local(),
						SensorData: map[string]float64{
							RegisteredQueryField:        23,
							AnotherRegisteredQueryField: 80,
						},
					},
					{
						DeviceID:  AnotherRegisteredDeviceId,
						Timestamp: AlsoInsideTimeRange.Local(),
						SensorData: map[string]float64{
							RegisteredQueryField:        25,
							AnotherRegisteredQueryField: 70,
						},
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
						DeviceID:  AnotherRegisteredDeviceId,
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
						{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
						{DeviceId: AnotherRegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
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
			},
			token: ValidToken,
			deviceRequests: data.BatchSensorDataRequest{
				Hardware: []data.Hardware{
					{
						DeviceIdParam: data.DeviceIdParam{DeviceId: RegisteredDeviceId},
						QueryFields:   []string{RegisteredQueryField, AnotherRegisteredQueryField},
					},
					{
						DeviceIdParam: data.DeviceIdParam{DeviceId: AnotherRegisteredDeviceId},
						QueryFields:   []string{RegisteredQueryField, AnotherRegisteredQueryField},
					},
				},
				TimeFrame: data.TimeFrame{
					Timezone: data.Timezone{ValidTimezone},
					Start:    RelativeStart,
				},
			},
		},

		"no data on all deviceids requested": {
			wantStatus: http.StatusNoContent,
			want:       data.BatchSensorDataResponse{},
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
						{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
						{DeviceId: AnotherRegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
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
			},
			token:          ValidToken,
			deviceRequests: DefaultBatchRequest(),
		},

		"no data on one of the deviceids requested": {
			wantStatus: http.StatusOK,
			want: data.BatchSensorDataResponse{
				Errors: []data.SensorDataError{
					{DeviceId: AnotherRegisteredDeviceId, Error: "No Data"},
				},
				Results: []data.SensorData{
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
						{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
						{DeviceId: AnotherRegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
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
			},
			token:          ValidToken,
			deviceRequests: DefaultBatchRequest(),
		},

		"no data due to bad connection": {
			wantErr:                 true,
			disconnectedDataFetcher: true,
			wantStatus:              http.StatusInternalServerError,
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
						{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
						{DeviceId: AnotherRegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
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
			},
			token:          ValidToken,
			deviceRequests: DefaultBatchRequest(),
		},

		"admin user can get sensor data from any company": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: data.BatchSensorDataResponse{
				Errors: []data.SensorDataError{},
				Results: []data.SensorData{
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
			MockApi: MockApi{
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
						DeviceID:  AnotherRegisteredDeviceId,
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
						{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
						{DeviceId: AnotherRegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
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
			},
			token:          ValidToken,
			deviceRequests: DefaultBatchRequest(),
		},

		"admin user can get sensor data from any network": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: data.BatchSensorDataResponse{
				Errors: []data.SensorDataError{},
				Results: []data.SensorData{
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
			MockApi: MockApi{
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
						DeviceID:  AnotherRegisteredDeviceId,
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
						{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
						{DeviceId: AnotherRegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
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
			},
			token:          ValidToken,
			deviceRequests: DefaultBatchRequest(),
		},

		"network user can get any sensor data from within network": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: data.BatchSensorDataResponse{
				Errors: []data.SensorDataError{},
				Results: []data.SensorData{
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
					{
						DeviceID:  AnotherRegisteredDeviceId,
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
						{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
						{DeviceId: AnotherRegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
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
			},
			token:          ValidToken,
			deviceRequests: DefaultBatchRequest(),
		},

		"network user cant get sensor data from other network": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: data.BatchSensorDataResponse{
				Errors: []data.SensorDataError{
					{
						DeviceId: RegisteredDeviceId,
						Error:    "Unauthorized access to this device.",
					},
					{
						DeviceId: AnotherRegisteredDeviceId,
						Error:    "Unauthorized access to this device.",
					},
				},
				Results: []data.SensorData{},
			},
			MockApi: MockApi{
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
						DeviceID:  AnotherRegisteredDeviceId,
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
						{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
						{DeviceId: AnotherRegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
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
			},
			token:          ValidToken,
			deviceRequests: DefaultBatchRequest(),
		},

		"user cant get sensor data from other company": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: data.BatchSensorDataResponse{
				Errors: []data.SensorDataError{
					{
						DeviceId: RegisteredDeviceId,
						Error:    "Unauthorized access to this device.",
					},
					{
						DeviceId: AnotherRegisteredDeviceId,
						Error:    "Unauthorized access to this device.",
					},
				},
				Results: []data.SensorData{},
			},
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
						DeviceID:  AnotherRegisteredDeviceId,
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
						{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
						{DeviceId: AnotherRegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
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
			},
			token:          ValidToken,
			deviceRequests: DefaultBatchRequest(),
		},

		"one successful request, one error": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: data.BatchSensorDataResponse{
				Errors: []data.SensorDataError{
					{
						DeviceId: AnotherRegisteredDeviceId,
						Error:    "Unauthorized access to this device.",
					},
				},
				Results: []data.SensorData{
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
						DeviceID:  AnotherRegisteredDeviceId,
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
						{DeviceId: AnotherRegisteredDeviceId, Company: AnotherRegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
						{DeviceId: AnotherRegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
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
			},
			token:          ValidToken,
			deviceRequests: DefaultBatchRequest(),
		},

		"unprocessable because asking for more than 5 hardware": {
			wantErr:    true,
			wantStatus: http.StatusUnprocessableEntity,
			want: data.BatchSensorDataResponse{
				Errors:  []data.SensorDataError{},
				Results: []data.SensorData{},
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
						DeviceID:  AnotherRegisteredDeviceId,
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
						{DeviceId: AnotherRegisteredDeviceId, Company: AnotherRegisteredCompany},
						{DeviceId: "device3", Company: RegisteredCompany},
						{DeviceId: "device4", Company: RegisteredCompany},
						{DeviceId: "device5", Company: RegisteredCompany},
						{DeviceId: "device6", Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
						{DeviceId: AnotherRegisteredDeviceId, Network: RegisteredNetwork},
						{DeviceId: "device3", Network: RegisteredNetwork},
						{DeviceId: "device4", Network: RegisteredNetwork},
						{DeviceId: "device5", Network: RegisteredNetwork},
						{DeviceId: "device6", Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField},
						}, {
							DeviceId:    AnotherRegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField},
						}, {
							DeviceId:    "device3",
							QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField},
						}, {
							DeviceId:    "device4",
							QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField},
						}, {
							DeviceId:    "device5",
							QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField},
						}, {
							DeviceId:    "device6",
							QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			token: ValidToken,
			deviceRequests: data.BatchSensorDataRequest{
				Hardware: []data.Hardware{
					{
						DeviceIdParam: data.DeviceIdParam{RegisteredDeviceId},
						QueryFields:   []string{"all"},
					},
					{
						DeviceIdParam: data.DeviceIdParam{AnotherRegisteredDeviceId},
						QueryFields:   []string{"all"},
					},
					{
						DeviceIdParam: data.DeviceIdParam{"device3"},
						QueryFields:   []string{"all"},
					},
					{
						DeviceIdParam: data.DeviceIdParam{"device4"},
						QueryFields:   []string{"all"},
					},
					{
						DeviceIdParam: data.DeviceIdParam{"device5"},
						QueryFields:   []string{"all"},
					},
					{
						DeviceIdParam: data.DeviceIdParam{"device6"},
						QueryFields:   []string{"all"},
					},
				},
				TimeFrame: data.TimeFrame{
					Start: RelativeStart,
				},
			},
		},

		"unknown token": {
			wantErr:        true,
			wantStatus:     http.StatusUnauthorized,
			token:          InvalidToken,
			want:           data.BatchSensorDataResponse{},
			deviceRequests: DefaultBatchRequest(),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			api, closeFunc := tc.MockApi.Setup(t)
			defer closeFunc()
			if tc.disconnectedDataFetcher {
				testFlux, err := data.NewTestingInflux("../../config.yml")
				require.Nil(t, err)
				testFlux.BadConnQueryApi()
				api.DataFetcher = testFlux
			}
			humaTest := setupHuma(t, api)
			route := "/batch/device/sensordata"
			resp := humaTest.Post(route,
				fmt.Sprintf(`Authorization: Bearer %s`, tc.token), tc.deviceRequests)
			if resp.Code != tc.wantStatus {
				t.Fatalf("wantStatus: %d, response status: %d", tc.wantStatus, resp.Code)
			}
			defer resp.Result().Body.Close()
			if !tc.wantErr && tc.wantStatus != http.StatusNoContent {
				var dd data.BatchSensorDataResponse
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

func TestBatchGetLatestSensorData(t *testing.T) {
	tests := map[string]struct {
		MockApi
		deviceRequests                   data.BatchLatestSensorDataRequest
		want                             data.BatchSensorDataResponse
		token                            string
		wantStatus                       int
		wantErr, disconnectedDataFetcher bool
	}{

		"get multiple deviceIds' latest data": {
			wantStatus: http.StatusOK,
			want: data.BatchSensorDataResponse{
				Errors: []data.SensorDataError{},
				Results: []data.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: AlsoInsideTimeRange,
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
							RegisteredQueryField:        30,
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
					{
						DeviceID:  AnotherRegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField:        10,
							AnotherRegisteredQueryField: 60,
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
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
						{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
						{DeviceId: AnotherRegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
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
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			token: ValidToken,
			deviceRequests: data.BatchLatestSensorDataRequest{
				Hardware: []data.Hardware{
					{
						DeviceIdParam: data.DeviceIdParam{RegisteredDeviceId},
						QueryFields: []string{
							RegisteredQueryField, AnotherRegisteredQueryField},
					},
					{
						DeviceIdParam: data.DeviceIdParam{AnotherRegisteredDeviceId},
						QueryFields: []string{
							RegisteredQueryField, AnotherRegisteredQueryField},
					},
				},
			},
		},

		"get multiple deviceIds' latest data in different timezone": {
			wantStatus: http.StatusOK,
			want: data.BatchSensorDataResponse{
				Errors: []data.SensorDataError{},
				Results: []data.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: AlsoInsideTimeRange.Local(),
						SensorData: map[string]float64{
							RegisteredQueryField:        23,
							AnotherRegisteredQueryField: 80,
						},
					},
					{
						DeviceID:  AnotherRegisteredDeviceId,
						Timestamp: AlsoInsideTimeRange.Local(),
						SensorData: map[string]float64{
							RegisteredQueryField:        25,
							AnotherRegisteredQueryField: 70,
						},
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
							RegisteredQueryField:        30,
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
					{
						DeviceID:  AnotherRegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField:        10,
							AnotherRegisteredQueryField: 60,
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
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
						{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
						{DeviceId: AnotherRegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
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
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			token: ValidToken,
			deviceRequests: data.BatchLatestSensorDataRequest{
				Timezone: data.Timezone{Timezone: ValidTimezone},
				Hardware: []data.Hardware{
					{
						DeviceIdParam: data.DeviceIdParam{RegisteredDeviceId},
						QueryFields: []string{
							RegisteredQueryField, AnotherRegisteredQueryField},
					},
					{
						DeviceIdParam: data.DeviceIdParam{AnotherRegisteredDeviceId},
						QueryFields: []string{
							RegisteredQueryField, AnotherRegisteredQueryField},
					},
				},
			},
		},

		"no data due to bad connection": {
			wantErr:                 true,
			wantStatus:              http.StatusInternalServerError,
			want:                    data.BatchSensorDataResponse{},
			disconnectedDataFetcher: true,
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
						{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
						{DeviceId: AnotherRegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
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
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			token: ValidToken,
			deviceRequests: data.BatchLatestSensorDataRequest{
				Hardware: []data.Hardware{
					{
						DeviceIdParam: data.DeviceIdParam{RegisteredDeviceId},
						QueryFields: []string{
							RegisteredQueryField, AnotherRegisteredQueryField},
					},
					{
						DeviceIdParam: data.DeviceIdParam{AnotherRegisteredDeviceId},
						QueryFields: []string{
							RegisteredQueryField, AnotherRegisteredQueryField},
					},
				},
			},
		},

		"no data on all deviceids requested": {
			wantStatus: http.StatusNoContent,
			want:       data.BatchSensorDataResponse{},
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
						{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
						{DeviceId: AnotherRegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
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
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			token: ValidToken,
			deviceRequests: data.BatchLatestSensorDataRequest{
				Hardware: []data.Hardware{
					{
						DeviceIdParam: data.DeviceIdParam{RegisteredDeviceId},
						QueryFields: []string{
							RegisteredQueryField, AnotherRegisteredQueryField},
					},
					{
						DeviceIdParam: data.DeviceIdParam{AnotherRegisteredDeviceId},
						QueryFields: []string{
							RegisteredQueryField, AnotherRegisteredQueryField},
					},
				},
			},
		},

		"no data on one of the deviceids requested": {
			wantStatus: http.StatusOK,
			want: data.BatchSensorDataResponse{
				Errors: []data.SensorDataError{
					{DeviceId: AnotherRegisteredDeviceId, Error: "No Data"},
				},
				Results: []data.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: AlsoInsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField:        23,
							AnotherRegisteredQueryField: 80,
						},
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
							RegisteredQueryField:        15,
							AnotherRegisteredQueryField: 45,
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
						{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
						{DeviceId: AnotherRegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
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
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			token: ValidToken,
			deviceRequests: data.BatchLatestSensorDataRequest{
				Hardware: []data.Hardware{
					{
						DeviceIdParam: data.DeviceIdParam{RegisteredDeviceId},
						QueryFields: []string{
							RegisteredQueryField, AnotherRegisteredQueryField},
					},
					{
						DeviceIdParam: data.DeviceIdParam{AnotherRegisteredDeviceId},
						QueryFields: []string{
							RegisteredQueryField, AnotherRegisteredQueryField},
					},
				},
			},
		},

		"admin user can get latest sensor data from any company": {
			wantStatus: http.StatusOK,
			want: data.BatchSensorDataResponse{
				Errors: []data.SensorDataError{},
				Results: []data.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: AlsoInsideTimeRange,
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
			MockApi: MockApi{
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
				mockDataFetcher: []data.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField:        18,
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
					{
						DeviceID:  AnotherRegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField:        13,
							AnotherRegisteredQueryField: 64,
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
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
						{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
						{DeviceId: AnotherRegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
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
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			token: ValidToken,
			deviceRequests: data.BatchLatestSensorDataRequest{
				Hardware: []data.Hardware{
					{
						DeviceIdParam: data.DeviceIdParam{DeviceId: RegisteredDeviceId},
						QueryFields: []string{
							RegisteredQueryField, AnotherRegisteredQueryField},
					},
					{
						DeviceIdParam: data.DeviceIdParam{AnotherRegisteredDeviceId},
						QueryFields: []string{
							RegisteredQueryField, AnotherRegisteredQueryField},
					},
				},
			},
		},

		"admin user can get latest sensor data from any network": {
			wantStatus: http.StatusOK,
			want: data.BatchSensorDataResponse{
				Errors: []data.SensorDataError{},
				Results: []data.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: AlsoInsideTimeRange,
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
			MockApi: MockApi{
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
				mockDataFetcher: []data.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField:        18,
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
					{
						DeviceID:  AnotherRegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField:        13,
							AnotherRegisteredQueryField: 64,
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
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
						{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
						{DeviceId: AnotherRegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
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
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			token: ValidToken,
			deviceRequests: data.BatchLatestSensorDataRequest{
				Hardware: []data.Hardware{
					{
						DeviceIdParam: data.DeviceIdParam{DeviceId: RegisteredDeviceId},
						QueryFields: []string{
							RegisteredQueryField, AnotherRegisteredQueryField},
					},
					{
						DeviceIdParam: data.DeviceIdParam{AnotherRegisteredDeviceId},
						QueryFields: []string{
							RegisteredQueryField, AnotherRegisteredQueryField},
					},
				},
			},
		},

		"network user can get any sensor data from within network": {
			wantStatus: http.StatusOK,
			want: data.BatchSensorDataResponse{
				Errors: []data.SensorDataError{},
				Results: []data.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: AlsoInsideTimeRange,
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
							RegisteredQueryField:        18,
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
					{
						DeviceID:  AnotherRegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField:        13,
							AnotherRegisteredQueryField: 64,
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
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
						{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
						{DeviceId: AnotherRegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
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
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			token: ValidToken,
			deviceRequests: data.BatchLatestSensorDataRequest{
				Hardware: []data.Hardware{
					{
						DeviceIdParam: data.DeviceIdParam{DeviceId: RegisteredDeviceId},
						QueryFields: []string{
							RegisteredQueryField, AnotherRegisteredQueryField},
					},
					{
						DeviceIdParam: data.DeviceIdParam{AnotherRegisteredDeviceId},
						QueryFields: []string{
							RegisteredQueryField, AnotherRegisteredQueryField},
					},
				},
			},
		},

		"network user cant get sensor data from other network": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: data.BatchSensorDataResponse{
				Errors: []data.SensorDataError{
					{
						DeviceId: RegisteredDeviceId,
						Error:    "Unauthorized access to this device.",
					},
					{
						DeviceId: AnotherRegisteredDeviceId,
						Error:    "Unauthorized access to this device.",
					},
				},
				Results: []data.SensorData{},
			},
			MockApi: MockApi{
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
						DeviceID:  AnotherRegisteredDeviceId,
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
						{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
						{DeviceId: AnotherRegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
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
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			token: ValidToken,
			deviceRequests: data.BatchLatestSensorDataRequest{
				Hardware: []data.Hardware{
					{
						DeviceIdParam: data.DeviceIdParam{DeviceId: RegisteredDeviceId},
						QueryFields: []string{
							RegisteredQueryField, AnotherRegisteredQueryField},
					},
					{
						DeviceIdParam: data.DeviceIdParam{AnotherRegisteredDeviceId},
						QueryFields: []string{
							RegisteredQueryField, AnotherRegisteredQueryField},
					},
				},
			},
		},

		"user cant get sensor data from other company": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: data.BatchSensorDataResponse{
				Errors: []data.SensorDataError{
					{
						DeviceId: RegisteredDeviceId,
						Error:    "Unauthorized access to this device.",
					},
					{
						DeviceId: AnotherRegisteredDeviceId,
						Error:    "Unauthorized access to this device.",
					},
				},
				Results: []data.SensorData{},
			},
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
						DeviceID:  AnotherRegisteredDeviceId,
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
						{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
						{DeviceId: AnotherRegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
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
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			token: ValidToken,
			deviceRequests: data.BatchLatestSensorDataRequest{
				Hardware: []data.Hardware{
					{
						DeviceIdParam: data.DeviceIdParam{DeviceId: RegisteredDeviceId},
						QueryFields: []string{
							RegisteredQueryField, AnotherRegisteredQueryField},
					},
					{
						DeviceIdParam: data.DeviceIdParam{AnotherRegisteredDeviceId},
						QueryFields: []string{
							RegisteredQueryField, AnotherRegisteredQueryField},
					},
				},
			},
		},

		"one successful request, one error": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: data.BatchSensorDataResponse{
				Errors: []data.SensorDataError{
					{
						DeviceId: AnotherRegisteredDeviceId,
						Error:    "Unauthorized access to this device.",
					},
				},
				Results: []data.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: AlsoInsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField:        23,
							AnotherRegisteredQueryField: 80,
						},
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
							RegisteredQueryField:        13,
							AnotherRegisteredQueryField: 60,
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
					{
						DeviceID:  AnotherRegisteredDeviceId,
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
						{DeviceId: AnotherRegisteredDeviceId, Company: AnotherRegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
						{DeviceId: AnotherRegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
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
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			token: ValidToken,
			deviceRequests: data.BatchLatestSensorDataRequest{
				Hardware: []data.Hardware{
					{
						DeviceIdParam: data.DeviceIdParam{DeviceId: RegisteredDeviceId},
						QueryFields: []string{
							RegisteredQueryField, AnotherRegisteredQueryField},
					},
					{
						DeviceIdParam: data.DeviceIdParam{AnotherRegisteredDeviceId},
						QueryFields: []string{
							RegisteredQueryField, AnotherRegisteredQueryField},
					},
				},
			},
		},

		"unprocessable because asking for more than 5 hardware": {
			wantErr:    true,
			wantStatus: http.StatusUnprocessableEntity,
			want: data.BatchSensorDataResponse{
				Errors:  []data.SensorDataError{},
				Results: []data.SensorData{},
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
						DeviceID:  AnotherRegisteredDeviceId,
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
						{DeviceId: AnotherRegisteredDeviceId, Company: AnotherRegisteredCompany},
						{DeviceId: "device3", Company: RegisteredCompany},
						{DeviceId: "device4", Company: RegisteredCompany},
						{DeviceId: "device5", Company: RegisteredCompany},
						{DeviceId: "device6", Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
						{DeviceId: AnotherRegisteredDeviceId, Network: RegisteredNetwork},
						{DeviceId: "device3", Network: RegisteredNetwork},
						{DeviceId: "device4", Network: RegisteredNetwork},
						{DeviceId: "device5", Network: RegisteredNetwork},
						{DeviceId: "device6", Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
						{
							DeviceId:    RegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField},
						}, {
							DeviceId:    AnotherRegisteredDeviceId,
							QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField},
						}, {
							DeviceId:    "device3",
							QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField},
						}, {
							DeviceId:    "device4",
							QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField},
						}, {
							DeviceId:    "device5",
							QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField},
						}, {
							DeviceId:    "device6",
							QueryFields: []string{RegisteredQueryField, AnotherRegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			token: ValidToken,
			deviceRequests: data.BatchLatestSensorDataRequest{
				Hardware: []data.Hardware{
					{
						DeviceIdParam: data.DeviceIdParam{RegisteredDeviceId},
						QueryFields:   []string{"all"},
					},
					{
						DeviceIdParam: data.DeviceIdParam{AnotherRegisteredDeviceId},
						QueryFields:   []string{"all"},
					},
					{
						DeviceIdParam: data.DeviceIdParam{"device3"},
						QueryFields:   []string{"all"},
					},
					{
						DeviceIdParam: data.DeviceIdParam{"device4"},
						QueryFields:   []string{"all"},
					},
					{
						DeviceIdParam: data.DeviceIdParam{"device5"},
						QueryFields:   []string{"all"},
					},
					{
						DeviceIdParam: data.DeviceIdParam{"device6"},
						QueryFields:   []string{"all"},
					},
				},
			},
		},

		"unknown token": {
			wantErr:    true,
			wantStatus: http.StatusUnauthorized,
			token:      InvalidToken,
			want:       data.BatchSensorDataResponse{},
			deviceRequests: data.BatchLatestSensorDataRequest{
				Hardware: []data.Hardware{
					{
						DeviceIdParam: data.DeviceIdParam{DeviceId: RegisteredDeviceId},
						QueryFields: []string{
							RegisteredQueryField, AnotherRegisteredQueryField},
					},
					{
						DeviceIdParam: data.DeviceIdParam{AnotherRegisteredDeviceId},
						QueryFields: []string{
							RegisteredQueryField, AnotherRegisteredQueryField},
					},
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			api, closeFunc := tc.MockApi.Setup(t)
			defer closeFunc()
			if tc.disconnectedDataFetcher {
				testFlux, err := data.NewTestingInflux("../../config.yml")
				require.Nil(t, err)
				testFlux.BadConnQueryApi()
				api.DataFetcher = testFlux
			}
			humaTest := setupHuma(t, api)
			route := "/batch/device/sensordata/latest"
			resp := humaTest.Post(route,
				fmt.Sprintf(`Authorization: Bearer %s`, tc.token), tc.deviceRequests)
			if resp.Code != tc.wantStatus {
				t.Fatalf("wantStatus: %d, response status: %d", tc.wantStatus, resp.Code)
			}
			defer resp.Result().Body.Close()
			if !tc.wantErr && tc.wantStatus != http.StatusNoContent {
				var dd data.BatchSensorDataResponse
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

func TestBatchGetQueryFields(t *testing.T) {
	tests := map[string]struct {
		MockApi
		want                            info.BatchQueryFieldsResponse
		queryFieldRequests              info.BatchQueryFieldsRequest
		token                           string
		wantStatus                      int
		wantErr, disconnectedDeviceInfo bool
	}{

		"get multiple deviceIds' queryfields": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: info.BatchQueryFieldsResponse{
				Errors: []info.QueryFieldsError{},
				Results: []info.QueryFields{
					{
						DeviceId:    RegisteredDeviceId,
						QueryFields: append(info.GeneralQueryFields, RegisteredQueryField),
					},
					{
						DeviceId: AnotherRegisteredDeviceId,
						QueryFields: append(info.GeneralQueryFields,
							AnotherRegisteredQueryField),
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
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
						{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
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
			token: ValidToken,
			queryFieldRequests: info.BatchQueryFieldsRequest{
				Body: device.Batch{
					DeviceIds: []string{
						RegisteredDeviceId,
						AnotherRegisteredDeviceId,
					},
				},
			},
		},

		"deviceIds not found": {
			wantErr:    true,
			wantStatus: http.StatusNotFound,
			want:       info.BatchQueryFieldsResponse{},
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
			token: ValidToken,
			queryFieldRequests: info.BatchQueryFieldsRequest{
				Body: device.Batch{
					DeviceIds: []string{
						AnotherRegisteredDeviceId,
						"Device3",
					},
				},
			},
		},

		"no data due to bad connection": {
			wantErr:                true,
			disconnectedDeviceInfo: true,
			wantStatus:             http.StatusInternalServerError,
			want:                   info.BatchQueryFieldsResponse{},
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
			token: ValidToken,
			queryFieldRequests: info.BatchQueryFieldsRequest{
				Body: device.Batch{
					DeviceIds: []string{
						RegisteredDeviceId,
					},
				},
			},
		},

		"admin user can get device queryfields from any company": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: info.BatchQueryFieldsResponse{
				Errors: []info.QueryFieldsError{},
				Results: []info.QueryFields{
					{
						DeviceId:    RegisteredDeviceId,
						QueryFields: append(info.GeneralQueryFields, RegisteredQueryField),
					},
					{
						DeviceId: AnotherRegisteredDeviceId,
						QueryFields: append(info.GeneralQueryFields,
							AnotherRegisteredQueryField),
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
						{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
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
			token: ValidToken,
			queryFieldRequests: info.BatchQueryFieldsRequest{
				Body: device.Batch{
					DeviceIds: []string{
						RegisteredDeviceId,
						AnotherRegisteredDeviceId,
					},
				},
			},
		},

		"admin user can get device queryfields from any network": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: info.BatchQueryFieldsResponse{
				Errors: []info.QueryFieldsError{},
				Results: []info.QueryFields{
					{
						DeviceId:    RegisteredDeviceId,
						QueryFields: append(info.GeneralQueryFields, RegisteredQueryField),
					},
					{
						DeviceId: AnotherRegisteredDeviceId,
						QueryFields: append(info.GeneralQueryFields,
							AnotherRegisteredQueryField),
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
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
						{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
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
			token: ValidToken,
			queryFieldRequests: info.BatchQueryFieldsRequest{
				Body: device.Batch{
					DeviceIds: []string{
						RegisteredDeviceId,
						AnotherRegisteredDeviceId,
					},
				},
			},
		},

		"network user can get any device queryfields from within network": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: info.BatchQueryFieldsResponse{
				Errors: []info.QueryFieldsError{},
				Results: []info.QueryFields{
					{
						DeviceId:    RegisteredDeviceId,
						QueryFields: append(info.GeneralQueryFields, RegisteredQueryField),
					},
					{
						DeviceId: AnotherRegisteredDeviceId,
						QueryFields: append(info.GeneralQueryFields,
							AnotherRegisteredQueryField),
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
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
						{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
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
			token: ValidToken,
			queryFieldRequests: info.BatchQueryFieldsRequest{
				Body: device.Batch{
					DeviceIds: []string{
						RegisteredDeviceId,
						AnotherRegisteredDeviceId,
					},
				},
			},
		},

		"network user cant get queryfields from other network": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: info.BatchQueryFieldsResponse{
				Errors: []info.QueryFieldsError{
					{
						DeviceId: RegisteredDeviceId,
						Error:    "Unauthorized access to this device.",
					},
					{
						DeviceId: AnotherRegisteredDeviceId,
						Error:    "Unauthorized access to this device.",
					},
				},
				Results: []info.QueryFields{},
			},
			MockApi: MockApi{
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
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
						{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
						{DeviceId: AnotherRegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
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
			},
			token: ValidToken,
			queryFieldRequests: info.BatchQueryFieldsRequest{
				Body: device.Batch{
					DeviceIds: []string{
						RegisteredDeviceId,
						AnotherRegisteredDeviceId,
					},
				},
			},
		},

		"user cant get device queryfields from other company": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: info.BatchQueryFieldsResponse{
				Errors: []info.QueryFieldsError{
					{
						DeviceId: RegisteredDeviceId,
						Error:    "Unauthorized access to this device.",
					},
					{
						DeviceId: AnotherRegisteredDeviceId,
						Error:    "Unauthorized access to this device.",
					},
				},
				Results: []info.QueryFields{},
			},
			MockApi: MockApi{
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
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
						{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
						{DeviceId: AnotherRegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
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
			},
			token: ValidToken,
			queryFieldRequests: info.BatchQueryFieldsRequest{
				Body: device.Batch{
					DeviceIds: []string{
						RegisteredDeviceId,
						AnotherRegisteredDeviceId,
					},
				},
			},
		},

		"one accessible deviceid, one inaccessible deviceid": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: info.BatchQueryFieldsResponse{
				Errors: []info.QueryFieldsError{
					{
						DeviceId: AnotherRegisteredDeviceId,
						Error:    "Unauthorized access to this device.",
					},
				},
				Results: []info.QueryFields{
					{
						DeviceId: RegisteredDeviceId,
						QueryFields: append(info.GeneralQueryFields,
							RegisteredQueryField),
					},
				},
			},
			MockApi: MockApi{
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
			token: ValidToken,
			queryFieldRequests: info.BatchQueryFieldsRequest{
				Body: device.Batch{
					DeviceIds: []string{
						RegisteredDeviceId, AnotherRegisteredDeviceId,
					},
				},
			},
		},

		"one valid deviceId, one invalid deviceId": {
			wantErr:    true,
			wantStatus: http.StatusUnprocessableEntity,
			want:       info.BatchQueryFieldsResponse{},
			MockApi: MockApi{
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
			token: ValidToken,
			queryFieldRequests: info.BatchQueryFieldsRequest{
				Body: device.Batch{
					DeviceIds: []string{
						RegisteredDeviceId, InvalidDeviceId,
					},
				},
			},
		},

		"unknown token": {
			wantErr:    true,
			wantStatus: http.StatusUnauthorized,
			token:      InvalidToken,
			want:       info.BatchQueryFieldsResponse{},
			queryFieldRequests: info.BatchQueryFieldsRequest{
				Body: device.Batch{
					DeviceIds: []string{
						RegisteredDeviceId, AnotherRegisteredDeviceId,
					},
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			api, closeFunc := tc.MockApi.Setup(t)
			defer closeFunc()
			if tc.disconnectedDeviceInfo {
				testFlux, err := data.NewTestingInflux("../../config.yml")
				require.Nil(t, err)
				testFlux.BadConnQueryApi()
				api.DataFetcher = testFlux
			}
			humaTest := setupHuma(t, api)
			route := "/batch/device/queryfields"
			resp := humaTest.Post(route,
				fmt.Sprintf(`Authorization: Bearer %s`, tc.token), tc.queryFieldRequests.Body)
			if resp.Code != tc.wantStatus {
				t.Fatalf("wantStatus: %d, response status: %d", tc.wantStatus, resp.Code)
			}
			defer resp.Result().Body.Close()
			if !tc.wantErr {
				var dd info.BatchQueryFieldsResponse
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

func TestBatchGetDataBoundary(t *testing.T) {
	tests := map[string]struct {
		MockApi
		want       data.BatchDataBoundaryResponse
		req        data.BatchDataBoundaryRequest
		token      string
		wantStatus int
		wantErr    bool
	}{

		"user get multiple deviceid data boundary": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: data.BatchDataBoundaryResponse{
				Errors: []data.BatchError{},
				Results: []data.DataBoundary{
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
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
						{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
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
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			token: ValidToken,
			req: data.BatchDataBoundaryRequest{
				Batch: device.Batch{
					DeviceIds: []string{RegisteredDeviceId, AnotherRegisteredDeviceId},
				},
			},
		},

		"no data for both deviceids": {
			wantStatus: http.StatusOK,
			want: data.BatchDataBoundaryResponse{
				Errors: []data.BatchError{
					{DeviceId: RegisteredDeviceId, Error: "No Data"},
					{DeviceId: AnotherRegisteredDeviceId, Error: "No Data"},
				},
				Results: []data.DataBoundary{},
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
				mockDataFetcher: []data.SensorData{},
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
						{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
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
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			token: ValidToken,
			req: data.BatchDataBoundaryRequest{
				Batch: device.Batch{
					DeviceIds: []string{RegisteredDeviceId, AnotherRegisteredDeviceId},
				},
			},
		},

		"no data for one of the deviceids request": {
			wantStatus: http.StatusOK,
			want: data.BatchDataBoundaryResponse{
				Errors: []data.BatchError{
					{DeviceId: AnotherRegisteredDeviceId, Error: "No Data"},
				},
				Results: []data.DataBoundary{
					{
						DeviceId: RegisteredDeviceId,
						Start:    InsideTimeRange,
						Stop:     AlsoInsideTimeRange,
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
						{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
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
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			token: ValidToken,
			req: data.BatchDataBoundaryRequest{
				Batch: device.Batch{
					DeviceIds: []string{RegisteredDeviceId, AnotherRegisteredDeviceId},
				},
			},
		},

		"user get multiple deviceid data boundary in specific timezone": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: data.BatchDataBoundaryResponse{
				Errors: []data.BatchError{},
				Results: []data.DataBoundary{
					{
						DeviceId: RegisteredDeviceId,
						Start:    InsideTimeRange.Local(),
						Stop:     AlsoInsideTimeRange.Local(),
					},
					{
						DeviceId: AnotherRegisteredDeviceId,
						Start:    InsideTimeRange.Local(),
						Stop:     AlsoInsideTimeRange.Local(),
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
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
						{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
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
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			token: ValidToken,
			req: data.BatchDataBoundaryRequest{
				Batch: device.Batch{
					DeviceIds: []string{RegisteredDeviceId, AnotherRegisteredDeviceId},
				},
				Timezone: data.Timezone{Timezone: "Africa/Johannesburg"},
			},
		},

		"network user get multiple deviceid data boundary": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: data.BatchDataBoundaryResponse{
				Errors: []data.BatchError{},
				Results: []data.DataBoundary{
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
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			token: ValidToken,
			req: data.BatchDataBoundaryRequest{
				Batch: device.Batch{
					DeviceIds: []string{RegisteredDeviceId, AnotherRegisteredDeviceId},
				},
			},
		},

		"admin user can get device queryfields from any network and company": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: data.BatchDataBoundaryResponse{
				Errors: []data.BatchError{},
				Results: []data.DataBoundary{
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
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
						{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
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
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			token: ValidToken,
			req: data.BatchDataBoundaryRequest{
				Batch: device.Batch{
					DeviceIds: []string{RegisteredDeviceId, AnotherRegisteredDeviceId},
				},
			},
		},

		"network user can get any device databoundary from within network": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: data.BatchDataBoundaryResponse{
				Errors: []data.BatchError{},
				Results: []data.DataBoundary{
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
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
						{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
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
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			token: ValidToken,
			req: data.BatchDataBoundaryRequest{
				Batch: device.Batch{
					DeviceIds: []string{RegisteredDeviceId, AnotherRegisteredDeviceId},
				},
			},
		},

		"network user cant get databoundary from other network": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: data.BatchDataBoundaryResponse{
				Results: []data.DataBoundary{},
				Errors: []data.BatchError{
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
			MockApi: MockApi{
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
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
						{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
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
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			token: ValidToken,
			req: data.BatchDataBoundaryRequest{
				Batch: device.Batch{
					DeviceIds: []string{RegisteredDeviceId, AnotherRegisteredDeviceId},
				},
			},
		},

		"user cant get device databoundary from other company": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: data.BatchDataBoundaryResponse{
				Results: []data.DataBoundary{},
				Errors: []data.BatchError{
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
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
						{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
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
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			token: ValidToken,
			req: data.BatchDataBoundaryRequest{
				Batch: device.Batch{
					DeviceIds: []string{RegisteredDeviceId, AnotherRegisteredDeviceId},
				},
			},
		},

		"one accessible deviceid, one inaccessible deviceid": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: data.BatchDataBoundaryResponse{
				Results: []data.DataBoundary{
					{
						DeviceId: RegisteredDeviceId,
						Start:    InsideTimeRange,
						Stop:     AlsoInsideTimeRange,
					},
				},
				Errors: []data.BatchError{
					{
						DeviceId: AnotherRegisteredDeviceId,
						Error:    "Unauthorized access to this device.",
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
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			token: ValidToken,
			req: data.BatchDataBoundaryRequest{
				Batch: device.Batch{
					DeviceIds: []string{RegisteredDeviceId, AnotherRegisteredDeviceId},
				},
			},
		},

		"unknown token": {
			wantErr:    true,
			wantStatus: http.StatusUnauthorized,
			token:      InvalidToken,
			want:       data.BatchDataBoundaryResponse{},
			req: data.BatchDataBoundaryRequest{
				Batch: device.Batch{
					DeviceIds: []string{RegisteredDeviceId, AnotherRegisteredDeviceId},
				},
			},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			api, closeFunc := tc.MockApi.Setup(t)
			defer closeFunc()
			humaTest := setupHuma(t, api)
			route := "/batch/device/databoundary"
			resp := humaTest.Post(route,
				fmt.Sprintf(`Authorization: Bearer %s`, tc.token), tc.req)
			if resp.Code != tc.wantStatus {
				t.Fatalf("wantStatus: %d, response status: %d", tc.wantStatus, resp.Code)
			}
			defer resp.Result().Body.Close()
			if !tc.wantErr {
				var got data.BatchDataBoundaryResponse
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

func TestBatchGetLocation(t *testing.T) {
	tests := map[string]struct {
		MockApi
		want       data.BatchLocationResponse
		req        data.BatchLocationRequest
		token      string
		wantStatus int
		wantErr    bool
	}{

		"user get multiple deviceid location": {
			wantStatus: http.StatusOK,
			want: data.BatchLocationResponse{
				Errors: []data.BatchError{},
				Results: []data.DeviceLocationResponse{
					{
						DeviceId:  RegisteredDeviceId,
						Time:      AlsoInsideTimeRange,
						Latitude:  Latitude,
						Longitude: Longitude,
					},
					{
						DeviceId:  AnotherRegisteredDeviceId,
						Time:      AlsoInsideTimeRange,
						Latitude:  AnotherLatitude,
						Longitude: AnotherLongitude,
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
							"latitude": Latitude, "longitude": Longitude,
						},
					},
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: AlsoInsideTimeRange,
						SensorData: map[string]float64{
							"latitude": Latitude, "longitude": Longitude,
						},
					},
					{
						DeviceID:  AnotherRegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							"latitude": AnotherLatitude, "longitude": AnotherLongitude,
						},
					},
					{
						DeviceID:  AnotherRegisteredDeviceId,
						Timestamp: AlsoInsideTimeRange,
						SensorData: map[string]float64{
							"latitude": AnotherLatitude, "longitude": AnotherLongitude,
						},
					},
				},
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
						{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
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
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			token: ValidToken,
			req: data.BatchLocationRequest{
				Batch: device.Batch{
					DeviceIds: []string{RegisteredDeviceId, AnotherRegisteredDeviceId},
				},
			},
		},

		"no data for both deviceids": {
			wantStatus: http.StatusOK,
			want: data.BatchLocationResponse{
				Errors: []data.BatchError{
					{DeviceId: RegisteredDeviceId, Error: "No Location"},
					{DeviceId: AnotherRegisteredDeviceId, Error: "No Location"},
				},
				Results: []data.DeviceLocationResponse{},
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
				mockDataFetcher: []data.SensorData{},
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
						{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
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
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			token: ValidToken,
			req: data.BatchLocationRequest{
				Batch: device.Batch{
					DeviceIds: []string{RegisteredDeviceId, AnotherRegisteredDeviceId},
				},
			},
		},

		"no data for one of the deviceids": {
			wantStatus: http.StatusOK,
			want: data.BatchLocationResponse{
				Errors: []data.BatchError{
					{DeviceId: AnotherRegisteredDeviceId, Error: "No Location"},
				},
				Results: []data.DeviceLocationResponse{
					{
						DeviceId:  RegisteredDeviceId,
						Time:      AlsoInsideTimeRange,
						Latitude:  Latitude,
						Longitude: Longitude,
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
							"latitude": Latitude, "longitude": Longitude,
						},
					},
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: AlsoInsideTimeRange,
						SensorData: map[string]float64{
							"latitude": Latitude, "longitude": Longitude,
						},
					},
				},
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
						{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
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
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			token: ValidToken,
			req: data.BatchLocationRequest{
				Batch: device.Batch{
					DeviceIds: []string{RegisteredDeviceId, AnotherRegisteredDeviceId},
				},
			},
		},

		"network user get multiple deviceid location": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: data.BatchLocationResponse{
				Errors: []data.BatchError{},
				Results: []data.DeviceLocationResponse{
					{
						DeviceId:  RegisteredDeviceId,
						Time:      AlsoInsideTimeRange,
						Latitude:  Latitude,
						Longitude: Longitude,
					},
					{
						DeviceId:  AnotherRegisteredDeviceId,
						Time:      AlsoInsideTimeRange,
						Latitude:  AnotherLatitude,
						Longitude: AnotherLongitude,
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
							"latitude": Latitude, "longitude": Longitude,
						},
					},
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: AlsoInsideTimeRange,
						SensorData: map[string]float64{
							"latitude": Latitude, "longitude": Longitude,
						},
					},
					{
						DeviceID:  AnotherRegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							"latitude": AnotherLatitude, "longitude": AnotherLongitude,
						},
					},
					{
						DeviceID:  AnotherRegisteredDeviceId,
						Timestamp: AlsoInsideTimeRange,
						SensorData: map[string]float64{
							"latitude": AnotherLatitude, "longitude": AnotherLongitude,
						},
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
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			token: ValidToken,
			req: data.BatchLocationRequest{
				Batch: device.Batch{
					DeviceIds: []string{RegisteredDeviceId, AnotherRegisteredDeviceId},
				},
			},
		},

		"admin user can get device queryfields from any network and company": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: data.BatchLocationResponse{
				Errors: []data.BatchError{},
				Results: []data.DeviceLocationResponse{
					{
						DeviceId: RegisteredDeviceId,
						Time:     AlsoInsideTimeRange,
						Latitude: Latitude, Longitude: Longitude,
					},
					{
						DeviceId: AnotherRegisteredDeviceId,
						Time:     AlsoInsideTimeRange,
						Latitude: AnotherLatitude, Longitude: AnotherLongitude,
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
							"latitude": Latitude, "longitude": Longitude,
						},
					},
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: AlsoInsideTimeRange,
						SensorData: map[string]float64{
							"latitude": Latitude, "longitude": Longitude,
						},
					},
					{
						DeviceID:  AnotherRegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							"longitude": AnotherLongitude, "latitude": AnotherLatitude,
						},
					},
					{
						DeviceID:  AnotherRegisteredDeviceId,
						Timestamp: AlsoInsideTimeRange,
						SensorData: map[string]float64{
							"longitude": AnotherLongitude, "latitude": AnotherLatitude,
						},
					},
				},
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
						{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
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
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			token: ValidToken,
			req: data.BatchLocationRequest{
				Batch: device.Batch{
					DeviceIds: []string{RegisteredDeviceId, AnotherRegisteredDeviceId},
				},
			},
		},

		"network user can get any device databoundary from within network": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: data.BatchLocationResponse{
				Errors: []data.BatchError{},
				Results: []data.DeviceLocationResponse{
					{
						DeviceId: RegisteredDeviceId,
						Time:     AlsoInsideTimeRange,
						Latitude: Latitude, Longitude: Longitude,
					},
					{
						DeviceId: AnotherRegisteredDeviceId,
						Time:     AlsoInsideTimeRange,
						Latitude: AnotherLatitude, Longitude: AnotherLongitude,
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
							"latitude": Latitude, "longitude": Longitude,
						},
					},
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: AlsoInsideTimeRange,
						SensorData: map[string]float64{
							"latitude": Latitude, "longitude": Longitude,
						},
					},
					{
						DeviceID:  AnotherRegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							"latitude": AnotherLatitude, "longitude": AnotherLongitude,
						},
					},
					{
						DeviceID:  AnotherRegisteredDeviceId,
						Timestamp: AlsoInsideTimeRange,
						SensorData: map[string]float64{
							"latitude": AnotherLatitude, "longitude": AnotherLongitude,
						},
					},
				},
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
						{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
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
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			token: ValidToken,
			req: data.BatchLocationRequest{
				Batch: device.Batch{
					DeviceIds: []string{RegisteredDeviceId, AnotherRegisteredDeviceId},
				},
			},
		},

		"network user cant get databoundary from other network": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: data.BatchLocationResponse{
				Results: []data.DeviceLocationResponse{},
				Errors: []data.BatchError{
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
			MockApi: MockApi{
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
				mockDataFetcher: []data.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							"latitude": Latitude, "longitude": Longitude,
						},
					},
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: AlsoInsideTimeRange,
						SensorData: map[string]float64{
							"latitude": Latitude, "longitude": Longitude,
						},
					},
					{
						DeviceID:  AnotherRegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							"latitude": AnotherLatitude, "longitude": AnotherLongitude,
						},
					},
					{
						DeviceID:  AnotherRegisteredDeviceId,
						Timestamp: AlsoInsideTimeRange,
						SensorData: map[string]float64{
							"latitude": AnotherLatitude, "longitude": AnotherLongitude,
						},
					},
				},
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
						{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
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
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			token: ValidToken,
			req: data.BatchLocationRequest{
				Batch: device.Batch{
					DeviceIds: []string{RegisteredDeviceId, AnotherRegisteredDeviceId},
				},
			},
		},

		"user cant get device databoundary from other company": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: data.BatchLocationResponse{
				Results: []data.DeviceLocationResponse{},
				Errors: []data.BatchError{
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
				mockDataFetcher: []data.SensorData{
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							"latitude": Latitude, "longitude": Longitude,
						},
					},
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: AlsoInsideTimeRange,
						SensorData: map[string]float64{
							"latitude": Latitude, "longitude": Longitude,
						},
					},
					{
						DeviceID:  AnotherRegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							"latitude": AnotherLatitude, "longitude": AnotherLongitude,
						},
					},
					{
						DeviceID:  AnotherRegisteredDeviceId,
						Timestamp: AlsoInsideTimeRange,
						SensorData: map[string]float64{
							"latitude": AnotherLatitude, "longitude": AnotherLongitude,
						},
					},
				},
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
						{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
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
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			token: ValidToken,
			req: data.BatchLocationRequest{
				Batch: device.Batch{
					DeviceIds: []string{RegisteredDeviceId, AnotherRegisteredDeviceId},
				},
			},
		},

		"one accessible deviceid, one inaccessible deviceid": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want: data.BatchLocationResponse{
				Results: []data.DeviceLocationResponse{
					{
						DeviceId: RegisteredDeviceId,
						Time:     AlsoInsideTimeRange,
						Latitude: Latitude, Longitude: Longitude,
					},
				},
				Errors: []data.BatchError{
					{
						DeviceId: AnotherRegisteredDeviceId,
						Error:    "Unauthorized access to this device.",
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
							"latitude": Latitude, "longitude": Longitude,
						},
					},
					{
						DeviceID:  RegisteredDeviceId,
						Timestamp: AlsoInsideTimeRange,
						SensorData: map[string]float64{
							"latitude": Latitude, "longitude": Longitude,
						},
					},
					{
						DeviceID:  AnotherRegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							"latitude": AnotherLatitude, "longitude": AnotherLongitude,
						},
					},
					{
						DeviceID:  AnotherRegisteredDeviceId,
						Timestamp: AlsoInsideTimeRange,
						SensorData: map[string]float64{
							"latitude": AnotherLatitude, "longitude": AnotherLongitude,
						},
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
							QueryFields: []string{RegisteredQueryField},
						},
					},
				},
				mockTokens: map[string]bool{
					ValidToken: true,
				},
			},
			token: ValidToken,
			req: data.BatchLocationRequest{
				Batch: device.Batch{
					DeviceIds: []string{RegisteredDeviceId, AnotherRegisteredDeviceId},
				},
			},
		},

		"unknown token": {
			wantErr:    true,
			wantStatus: http.StatusUnauthorized,
			token:      InvalidToken,
			want:       data.BatchLocationResponse{},
			req: data.BatchLocationRequest{
				device.Batch{
					DeviceIds: []string{RegisteredDeviceId, AnotherRegisteredDeviceId},
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			api, closeFunc := tc.MockApi.Setup(t)
			defer closeFunc()
			humaTest := setupHuma(t, api)
			route := "/batch/device/location"
			resp := humaTest.Post(route,
				fmt.Sprintf(`Authorization: Bearer %s`, tc.token), tc.req)
			if resp.Code != tc.wantStatus {
				t.Fatalf("wantStatus: %d, response status: %d", tc.wantStatus, resp.Code)
			}
			defer resp.Result().Body.Close()
			if !tc.wantErr {
				var got data.BatchLocationResponse
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

func TestBatchCsvGetSensorData(t *testing.T) {
	tests := map[string]struct {
		MockApi
		GetSensorDataTest
		want string
	}{

		"multiple deviceid, single queryfield": {
			want: fmt.Sprintf(",%s\n%s\n%s,%s\n%s\n%s,%s\n",
				RegisteredQueryField, RegisteredDeviceId,
				InsideTimeRange.Format(time.RFC3339), "23.000",
				AnotherRegisteredDeviceId, InsideTimeRange.Format(time.RFC3339),
				"25.000"),
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
						DeviceID:  AnotherRegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField: 25,
						},
					},
				},
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
						{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
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
				BatchSensorDataRequest: &data.BatchSensorDataRequest{
					Hardware: []data.Hardware{
						{
							DeviceIdParam: data.DeviceIdParam{RegisteredDeviceId},
							QueryFields:   []string{RegisteredQueryField},
						},
						{
							DeviceIdParam: data.DeviceIdParam{AnotherRegisteredDeviceId},
							QueryFields:   []string{RegisteredQueryField},
						},
					},
					TimeFrame: data.TimeFrame{
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
						DeviceID:  AnotherRegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField: 25,
						},
					},
				},
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
						{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
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
				BatchSensorDataRequest: &data.BatchSensorDataRequest{
					Hardware: []data.Hardware{
						{
							DeviceIdParam: data.DeviceIdParam{RegisteredDeviceId},
							QueryFields:   []string{RegisteredQueryField},
						},
						{
							DeviceIdParam: data.DeviceIdParam{AnotherRegisteredDeviceId},
							QueryFields:   []string{RegisteredQueryField},
						},
					},
					TimeFrame: data.TimeFrame{
						Start:    RelativeStart,
						Timezone: data.Timezone{Timezone: "Africa/Johannesburg"},
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
						DeviceID:  AnotherRegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField:        25,
							AnotherRegisteredQueryField: 81,
						},
					},
				},
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
						{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
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
				BatchSensorDataRequest: &data.BatchSensorDataRequest{
					Hardware: []data.Hardware{
						{
							DeviceIdParam: data.DeviceIdParam{RegisteredDeviceId},
							QueryFields: []string{
								RegisteredQueryField, AnotherRegisteredQueryField},
						},
						{
							DeviceIdParam: data.DeviceIdParam{AnotherRegisteredDeviceId},
							QueryFields: []string{
								RegisteredQueryField, AnotherRegisteredQueryField},
						},
					},
					TimeFrame: data.TimeFrame{
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
						DeviceID:  AnotherRegisteredDeviceId,
						Timestamp: AlsoInsideTimeRange,
						SensorData: map[string]float64{
							AnotherRegisteredQueryField: 81,
						},
					},
				},
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
						{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
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
				BatchSensorDataRequest: &data.BatchSensorDataRequest{
					Hardware: []data.Hardware{
						{
							DeviceIdParam: data.DeviceIdParam{RegisteredDeviceId},
							QueryFields: []string{
								RegisteredQueryField, AnotherRegisteredQueryField},
						},
						{
							DeviceIdParam: data.DeviceIdParam{AnotherRegisteredDeviceId},
							QueryFields: []string{
								RegisteredQueryField, AnotherRegisteredQueryField},
						},
					},
					TimeFrame: data.TimeFrame{
						Start: RelativeStart,
					},
				},
			},
		},

		"multiple deviceid, all errors": {
			want: fmt.Sprintf("%s,%s\n%s,%s\n", RegisteredDeviceId,
				"Unauthorized access to this device.", AnotherRegisteredDeviceId, "Unauthorized access to this device."),
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
				mockDataFetcher: []data.SensorData{
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
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: RegisteredCompany},
						{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
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
				BatchSensorDataRequest: &data.BatchSensorDataRequest{
					Hardware: []data.Hardware{
						{
							DeviceIdParam: data.DeviceIdParam{RegisteredDeviceId},
							QueryFields:   []string{RegisteredQueryField},
						},
						{
							DeviceIdParam: data.DeviceIdParam{AnotherRegisteredDeviceId},
							QueryFields:   []string{RegisteredQueryField},
						},
					},
					TimeFrame: data.TimeFrame{
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
				mockDataFetcher: []data.SensorData{
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
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: AnotherRegisteredCompany},
						{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
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
				BatchSensorDataRequest: &data.BatchSensorDataRequest{
					Hardware: []data.Hardware{
						{
							DeviceIdParam: data.DeviceIdParam{RegisteredDeviceId},
							QueryFields:   []string{RegisteredQueryField},
						},
						{
							DeviceIdParam: data.DeviceIdParam{AnotherRegisteredDeviceId},
							QueryFields:   []string{RegisteredQueryField},
						},
					},
					TimeFrame: data.TimeFrame{
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
						DeviceID:  AnotherRegisteredDeviceId,
						Timestamp: InsideTimeRange,
						SensorData: map[string]float64{
							RegisteredQueryField: 25,
						},
					},
				},
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: AnotherRegisteredCompany},
						{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
						{DeviceId: AnotherRegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
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
			},
			GetSensorDataTest: GetSensorDataTest{wantErr: false,
				wantStatus: http.StatusOK,
				token:      ValidToken,
				BatchSensorDataRequest: &data.BatchSensorDataRequest{
					Hardware: []data.Hardware{
						{
							DeviceIdParam: data.DeviceIdParam{RegisteredDeviceId},
							QueryFields: []string{
								RegisteredQueryField, AnotherRegisteredQueryField},
						},
						{
							DeviceIdParam: data.DeviceIdParam{AnotherRegisteredDeviceId},
							QueryFields:   []string{RegisteredQueryField},
						},
					},
					TimeFrame: data.TimeFrame{
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
							RegisteredQueryField: 23,
						},
					},
				},
				mockDeviceInfo: device.Schema{
					DeviceCompanies: []device.DeviceToCompany{
						{DeviceId: RegisteredDeviceId, Company: AnotherRegisteredCompany},
						{DeviceId: AnotherRegisteredDeviceId, Company: RegisteredCompany},
					},
					DeviceNetworks: []device.DeviceToNetwork{
						{DeviceId: RegisteredDeviceId, Network: RegisteredNetwork},
						{DeviceId: AnotherRegisteredDeviceId, Network: RegisteredNetwork},
					},
					DeviceToQF: []device.DeviceToQueryFields{
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
			},
			GetSensorDataTest: GetSensorDataTest{wantErr: false,
				wantStatus: http.StatusOK,
				token:      ValidToken,
				BatchSensorDataRequest: &data.BatchSensorDataRequest{
					Hardware: []data.Hardware{
						{
							DeviceIdParam: data.DeviceIdParam{RegisteredDeviceId},
							QueryFields: []string{
								RegisteredQueryField, AnotherRegisteredQueryField},
						},
						{
							DeviceIdParam: data.DeviceIdParam{AnotherRegisteredDeviceId},
							QueryFields:   []string{RegisteredQueryField},
						},
					},
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
			route := "/batch/device/sensordata"
			resp := humaTest.Post(route,
				fmt.Sprintf(`Authorization: Bearer %s`, tc.token),
				`Accept: text/csv`,
				tc.BatchSensorDataRequest,
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
