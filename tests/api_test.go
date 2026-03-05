package tests

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/geraud22/datafarm-api/api"
	"github.com/geraud22/datafarm-api/authoriser"
	"github.com/geraud22/datafarm-api/datafetcher"
	mdf "github.com/geraud22/datafarm-api/metadatafetcher"
	"github.com/geraud22/datafarm-api/redis"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

const RegisteredUsername = "user1"
const UnregisteredUsername = "user2"
const RegisteredPassword = "@Password1"
const UnregisteredPassword = "@Password2"
const RegisteredCompany = "company"
const RegisteredNetwork = "network"
const UserRole = "1"

var a = &api.Api{}

func TestLogin(t *testing.T) {
	tests := map[string]struct {
		wantErr            bool
		wantStatus         int
		username, password string
		mockBasicAuth      map[string]authoriser.UserInfo
		mdfSchema, wantMdf mdf.Schema
	}{

		"successfully login": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			username:   RegisteredUsername,
			password:   RegisteredPassword,
			mockBasicAuth: map[string]authoriser.UserInfo{
				RegisteredUsername: {
					Username: RegisteredUsername,
					Company:  RegisteredCompany,
					Role:     UserRole,
					Password: RegisteredPassword,
					Network:  RegisteredNetwork,
				},
			},
		},

		"deny access": {
			wantErr:    true,
			wantStatus: http.StatusUnauthorized,
			username:   UnregisteredUsername,
			password:   UnregisteredPassword,
			mockBasicAuth: map[string]authoriser.UserInfo{
				RegisteredUsername: {
					Username: RegisteredUsername,
					Company:  RegisteredCompany,
					Role:     UserRole,
					Password: RegisteredPassword,
					Network:  RegisteredNetwork,
				},
			},
		},
	}

	var err error
	_, humaApi := humatest.New(t)
	a.RegisterHumaOperations(humaApi)
	db, err := miniredis.Run()
	require.Nil(t, err)
	defer db.Close()
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			a.TokenAuth = &authoriser.MockTokenAuth{}
			defer a.TokenAuth.Close()
			testingRedis, err := redis.NewTestingRedis(db.Addr())
			require.Nil(t, err)
			defer testingRedis.Close()
			a.MetadataFetcher = testingRedis
			a.BasicAuth = testingRedis
			err = testingRedis.PrepareMetadataFetcher(tc.mdfSchema)
			require.Nil(t, err)
			err = testingRedis.PrepareBasicAuth(tc.mockBasicAuth)
			require.Nil(t, err)
			encodedDetails := base64.StdEncoding.EncodeToString(
				[]byte(tc.username + ":" + tc.password))
			resp := humaApi.Post("/login",
				fmt.Sprintf("Authorization: Basic %s", encodedDetails))
			if resp.Code != tc.wantStatus {
				t.Fatalf("wantStatus: %d, response status: %d", tc.wantStatus, resp.Code)
			}
			if !tc.wantErr {
				schema := a.MetadataFetcher.GetSnapshot()
				if len(schema.UserTokens) != 1 {
					t.Fatalf("expected a stored user token, got len: %d", len(schema.UserTokens))
				}
			}
		})
	}
}

func TestGetDeviceData(t *testing.T) {
	tests := map[string]struct {
		wantErr         bool
		wantStatus      int
		want            *datafetcher.ConsolidatedDeviceData
		mockBasicAuth   map[string]authoriser.UserInfo
		mockDataFetcher any
		mdfSchema       mdf.Schema
		token           string
		deviceRequest   datafetcher.DeviceDataRequest
	}{

		"successfully get deviceid data": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			want:       &datafetcher.ConsolidatedDeviceData{},
			mockBasicAuth: map[string]authoriser.UserInfo{
				RegisteredUsername: {
					Username: RegisteredUsername,
					Company:  RegisteredCompany,
					Role:     UserRole,
					Password: RegisteredPassword,
					Network:  RegisteredNetwork,
				},
			},
			mockDataFetcher: nil,
			mdfSchema:       mdf.Schema{},
			token:           "",
			deviceRequest:   datafetcher.DeviceDataRequest{},
		},

		"invalid token": {},

		"expired token": {},
	}

	var err error
	_, humaApi := humatest.New(t)
	a.RegisterHumaOperations(humaApi)
	db, err := miniredis.Run()
	require.Nil(t, err)
	defer db.Close()
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			a.TokenAuth = &authoriser.MockTokenAuth{}
			defer a.TokenAuth.Close()
			testingRedis, err := redis.NewTestingRedis(db.Addr())
			require.Nil(t, err)
			defer testingRedis.Close()
			a.MetadataFetcher = testingRedis
			a.BasicAuth = testingRedis
			a.DataFetcher, err = datafetcher.NewTestingInflux("../config.yml")
			require.Nil(t, err)
			defer a.DataFetcher.Close()
			err = a.DataFetcher.PrepareDb(tc.mockDataFetcher)
			require.Nil(t, err)
			err = testingRedis.PrepareMetadataFetcher(tc.mdfSchema)
			require.Nil(t, err)
			err = testingRedis.PrepareBasicAuth(tc.mockBasicAuth)
			require.Nil(t, err)
			route := "/device/" + tc.deviceRequest.DeviceId
			resp := humaApi.Post(route,
				fmt.Sprintf("Authorization: Bearer %s", tc.token), tc.deviceRequest)
			if resp.Code != tc.wantStatus {
				t.Fatalf("wantStatus: %d, response status: %d", tc.wantStatus, resp.Code)
			}
			defer resp.Result().Body.Close()
			body, err := io.ReadAll(resp.Body)
			require.Nil(t, err)
			var cdd *datafetcher.ConsolidatedDeviceData
			err = json.Unmarshal(body, cdd)
			require.Nil(t, err)
			if !tc.wantErr {
				if diff := cmp.Diff(tc.want, cdd); diff != "" {
					t.Fatalf("response mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
