package tokenprovider

import (
	"github.com/datafarm-software/datafarm-api/api/authstore"
	"github.com/datafarm-software/datafarm-api/api/telemetry/logging"
)

type LoginRequest struct {
	Auth string `header:"Authorization" required:"true" hidden:"true"`
}

func (l *LoginRequest) Metadata() (m logging.Metadata) {
	m.KeyValue = make(map[string]string, 1)
	m.KeyValue["loginrequest.auth_header"] = l.Auth
	return
}

type LoginResponse struct {
	Body string `doc:"Access token for API resources. Three Hour Expiry."`
}

type TokenProvider interface {
	Close() error
	GenerateToken(username string) (authstore.UserToken, error)
	ValidToken(LoginResponse) bool
}
