package tokenprovider

import "time"

type LoginRequest struct {
	Auth string `header:"Authorization" required:"true" hidden:"true"`
}
type LoginResponse struct {
	Body string `doc:"Access token for API resources."`
}

type TokenProvider interface {
	Close() error
	GenerateToken() (string, time.Duration, error)
	IsValidToken(LoginResponse) bool
}
