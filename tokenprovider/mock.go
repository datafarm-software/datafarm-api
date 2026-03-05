package tokenprovider

import (
	"fmt"
	"time"
)

type MockTokenProvider struct{}

func (m *MockTokenProvider) Close() error {
	return nil
}

var increment int

func (m *MockTokenProvider) GenerateToken() (string, time.Duration, error) {
	token := fmt.Sprintf("someToken%d", increment)
	increment++
	return token, THREE_HOURS, nil
}

func (m *MockTokenProvider) IsValidToken(TokenResponse) bool {
	return false
}

type MockBasicAuth struct {
	db map[string]UserInfo
}

func (m *MockBasicAuth) PrepareDb(db map[string]UserInfo) error {
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
