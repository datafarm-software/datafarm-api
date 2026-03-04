package authoriser

import "time"

type UserInfo struct {
	Username string `redis:"username"`
	Company  string `redis:"company"`
	Role     string `redis:"role"`
	Password string `redis:"password"`
	Network  string
}

type UserToken struct {
	Username   string
	Token      string
	Expiration time.Duration
}

type TokenResponse struct {
	Token string `doc:"Access token for API resources."`
}

type TokenAuth interface {
	Close() error
	GenerateToken() (string, time.Duration, error)
	// GetPublicKey() *ecdsa.PublicKey
}

type TestBasicAuth interface {
	PrepareBasicAuth(map[string]UserInfo) error
}

type BasicAuth interface {
	TestBasicAuth
	Close() error
	VerifyCredentials(username, passw string) error
}
