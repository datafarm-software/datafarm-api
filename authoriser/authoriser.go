package authoriser

type UserInfo struct {
	Username string `redis:"username"`
	Company  string `redis:"company"`
	Role     string `redis:"role"`
	Password string `redis:"password"`
	Network  string
}

type TokenAuth interface {
	Close() error
	GenerateToken() (string, error)
	// GetPublicKey() *ecdsa.PublicKey
}

type TestBasicAuth interface {
	PrepareDb(map[string]UserInfo) error
}

type BasicAuth interface {
	TestBasicAuth
	Close() error
	VerifyCredentials(username, passw string) error
}
