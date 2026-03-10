package tokenprovider

import "time"

type TokenResponse struct {
	Body string `doc:"Access token for API resources."`
}

type TokenProvider interface {
	Close() error
	GenerateToken() (string, time.Duration, error)
	IsValidToken(TokenResponse) bool
}
