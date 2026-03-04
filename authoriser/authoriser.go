package authoriser

import "crypto/ecdsa"

type UserInfo struct {
	Username string `redis:"username"`
	Company  string `redis:"company"`
	Role     string `redis:"role"`
	Password string `redis:"password"`
	Network  string
}

type TokenAuth interface {
	Close() error
	GenerateToken(userInfo UserInfo) (string, error)
	GetPublicKey() *ecdsa.PublicKey
}

type BasicAuth interface {
	Close() error
	CheckCredentials(username, passw string) (UserInfo, error)
}
