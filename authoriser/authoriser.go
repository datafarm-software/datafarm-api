package authoriser

import (
	"time"
)

type Schema struct {
	UserInfo   map[string]UserInfo
	UserTokens []UserToken
}

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

type TokenProvider interface {
	Close() error
	GenerateToken() (string, time.Duration, error)
	IsValidToken(TokenResponse) bool
}

type TestAuthStore interface {
	PrepareAuthStore(Schema) error
	GetActiveTokens() []UserToken
}

type AuthStore interface {
	TestAuthStore
	Close() error
	VerifyCredentials(username, passw string) error
	GetUser(token string) (UserInfo, error)
	StoreToken(UserToken) error
	DeleteToken(TokenResponse) error
}
