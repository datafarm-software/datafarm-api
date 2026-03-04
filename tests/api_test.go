package tests

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/geraud22/datafarm-api/api"
	"github.com/geraud22/datafarm-api/authoriser"
)

const RegisteredUsername = "user1"
const UnregisteredUsername = "user2"
const RegisteredPassword = "pass1"
const UnregisteredPassword = "pass2"

var a = &api.Api{}

func TestLogin(t *testing.T) {
	tests := map[string]struct {
		wantErr            bool
		wantStatus         int
		username, password string
	}{
		"successfully login": {
			wantErr:    false,
			wantStatus: http.StatusOK,
			username:   RegisteredUsername,
			password:   RegisteredPassword,
		},

		"deny access": {
			wantErr:    true,
			wantStatus: http.StatusUnauthorized,
			username:   UnregisteredUsername,
			password:   UnregisteredPassword,
		},
	}

	_, humaApi := humatest.New(t)
	a.RegisterHumaOperations(humaApi)
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			a.TokenAuth = &authoriser.MockTokenAuth{}
			defer a.TokenAuth.Close()
			a.BasicAuth = &authoriser.MockBasicAuth{}
			defer a.BasicAuth.Close()
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
