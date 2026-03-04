package authoriser

import (
	"crypto/ecdsa"
	"fmt"
)

type MockTokenAuth struct{}

func (m *MockTokenAuth) Close() error {
	return nil
}

func (m *MockTokenAuth) GenerateToken(userInfo UserInfo) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (m *MockTokenAuth) GetPublicKey() *ecdsa.PublicKey {
	return nil
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

func (m *MockBasicAuth) CheckCredentials(username, passw string) (UserInfo, error) {
	userInfo, ok := m.db[username]
	if !ok {
		return UserInfo{}, fmt.Errorf("no user info")
	}
	return userInfo, nil
}
