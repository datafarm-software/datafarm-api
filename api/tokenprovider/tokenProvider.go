package tokenprovider

import (
	"github.com/datafarm-software/datafarm-api/api/authstore"
)

type LoginRequest struct {
	Auth string `header:"Authorization" required:"true" hidden:"true"`
}
type LoginResponse struct {
	Body string `doc:"Access token for API resources. Three Hour Expiry."`
}

type TokenProvider interface {
	Close() error
	GenerateToken(username string) (authstore.UserToken, error)
	IsValidToken(LoginResponse) bool
}
