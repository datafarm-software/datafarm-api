package authoriser

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"io/fs"
)

type JwtAuth struct {
	publicKey *ecdsa.PublicKey
}

func NewJwtAuth(fs fs.FS, publicKeyPath string) (*JwtAuth, error) {
	if fs == nil || publicKeyPath == "" {
		return nil, fmt.Errorf("not all options present")
	}
	var jwtAuth JwtAuth
	publicKey, err := jwtAuth.loadECDSAPublicKey(fs, publicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("error loading public key: %v", err)
	}
	jwtAuth.publicKey = publicKey
	return &jwtAuth, nil
}

func (j *JwtAuth) loadECDSAPublicKey(fs fs.FS, filePath string) (*ecdsa.PublicKey, error) {
	file, err := fs.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key file: %w", err)
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key file: %w", err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}
	pubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ECDSA public key: %w", err)
	}
	ecdsaPubKey, ok := pubKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("invalid ECDSA public key")
	}
	return ecdsaPubKey, nil
}

func (j *JwtAuth) GetPublicKey() *ecdsa.PublicKey {
	return j.publicKey
}

func (j *JwtAuth) GenerateJwt() (string, error) {
	return "", nil
}
