package authstore

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
	DeleteToken(UserToken) error
}
