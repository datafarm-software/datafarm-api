package authstore

import (
	"errors"
	"time"
)

var NotLoggedIn = errors.New("not logged in.")

type Schema struct {
	UserInfo   []UserInfo
	UserTokens []UserToken
}

type UserInfo struct {
	Username string `redis:"username"`
	Company  string `redis:"company"`
	Role     int    `redis:"role"`
	Password string `redis:"password"`
	Network  string `redis:"network"`
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
	//NOTE: could return err: NotLoggedIn
	GetToken(username string) (UserToken, error)
}
