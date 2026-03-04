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

type MockBasicAuth struct{}

func (m *MockBasicAuth) Close() error {
	return nil
}

func (m *MockBasicAuth) CheckCredentials(username, passw string) (UserInfo, error) {
	return UserInfo{}, fmt.Errorf("not implemented")
}
