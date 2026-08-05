package tokenprovider

import (
	"fmt"

	"github.com/datafarm-software/datafarm-api/api/authstore"
)

type MockTokenProvider struct {
	Tokens    map[string]bool
	Increment int
}

func (m *MockTokenProvider) Close() error {
	return nil
}

func (m *MockTokenProvider) GenerateToken(username string) (authstore.UserToken, error) {
	token := fmt.Sprintf("someToken%d", m.Increment)
	m.Increment++
	if m.Tokens == nil {
		m.Tokens = make(map[string]bool)
	}
	m.Tokens[token] = true
	return authstore.UserToken{
		Username:   username,
		Token:      token,
		Expiration: THREE_HOURS,
	}, nil
}

func (m *MockTokenProvider) IsValidToken(t LoginResponse) bool {
	return m.Tokens[t.Body]
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
