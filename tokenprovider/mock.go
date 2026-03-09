package tokenprovider

import (
	"fmt"
	"time"

	"github.com/geraud22/datafarm-api/authstore"
)

type MockTokenProvider struct {
	Tokens    map[string]bool
	Increment int
}

func (m *MockTokenProvider) Close() error {
	return nil
}

func (m *MockTokenProvider) GenerateToken() (string, time.Duration, error) {
	token := fmt.Sprintf("someToken%d", m.Increment)
	m.Increment++
	m.Tokens[token] = true
	return token, THREE_HOURS, nil
}

func (m *MockTokenProvider) IsValidToken(t TokenResponse) bool {
	return m.Tokens[t.Token]
}

type MockBasicAuth struct {
	db map[string]authstore.UserInfo
}

func (m *MockBasicAuth) PrepareDb(db map[string]authstore.UserInfo) error {
	m.db = db
	return nil
}

func (m *MockBasicAuth) Close() error {
	return nil
}

func (m *MockBasicAuth) VerifyCredentials(username, passw string) error {
	var err error
	if _, ok := m.db[username]; !ok {
		err = fmt.Errorf("no user info")
	}
	return err
}
