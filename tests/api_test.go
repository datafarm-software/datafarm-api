package tests

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/geraud22/datafarm-api/api"
	"github.com/geraud22/datafarm-api/authoriser"
	"github.com/geraud22/datafarm-api/redis"
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
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			a.TokenAuth = &authoriser.MockTokenAuth{}
			defer a.TokenAuth.Close()
			testingRedis, err := redis.NewTestingRedis(db.Addr())
			require.Nil(t, err)
			defer testingRedis.Close()
			a.MetadataFetcher = testingRedis
			a.BasicAuth = testingRedis
			err = testingRedis.PrepareMetadataFetcher(tc.mockMdf)
			require.Nil(t, err)
			err = testingRedis.PrepareBasicAuth(tc.mockBasicAuth)
			require.Nil(t, err)
			byteDetails := bytes.NewBuffer(nil)
			fmt.Fprintf(byteDetails, "%s:%s", tc.username, tc.password)
			encodedDetails := base64.StdEncoding.EncodeToString(byteDetails.Bytes())
			resp := humaApi.Post("/login",
				fmt.Sprintf("Authorization: Basic %s", encodedDetails))
			if resp.Code != tc.wantStatus {
				t.Fatalf("wantStatus: %d, response status: %d", tc.wantStatus, resp.Code)
			}
		})
	}
}
