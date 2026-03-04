package authoriser

import (
	"crypto/ecdsa"
	"fmt"
)

type MockTokenAuth struct{}

func (m *MockTokenAuth) Close() error {
	return nil
}

func (m *MockTokenAuth) GenerateToken() (string, error) {
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

func (m *MockBasicAuth) VerifyCredentials(username, passw string) error {
	var err error
	if _, ok := m.db[username]; !ok {
		err = fmt.Errorf("no user info")
	}
	return err
}
