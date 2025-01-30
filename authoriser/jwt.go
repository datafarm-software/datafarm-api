package authoriser

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"io/fs"
	"time"

	"github.com/golang-jwt/jwt"
)

type JwtAuth struct {
	fs         fs.FS
	publicKey  *ecdsa.PublicKey
	privateKey *ecdsa.PrivateKey
}

func NewJwtAuth(fs fs.FS, privateKeyPath, publicKeyPath string) (*JwtAuth, error) {
	if fs == nil || publicKeyPath == "" {
		return nil, fmt.Errorf("not all options present")
	}
	var jwtAuth JwtAuth
	jwtAuth.fs = fs
	publicKey, err := jwtAuth.loadECDSAPublicKey(publicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("error loading public key: %v", err)
	}
	jwtAuth.publicKey = publicKey
	privateKey, err := jwtAuth.loadECDSAPrivateKey(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("error loading public key: %v", err)
	}
	jwtAuth.privateKey = privateKey
	return &jwtAuth, nil
}

func (j *JwtAuth) loadECDSAPublicKey(filePath string) (*ecdsa.PublicKey, error) {
	file, err := j.fs.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key file: %w", err)
	}
	defer file.Close()
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

func (j *JwtAuth) loadECDSAPrivateKey(filePath string) (*ecdsa.PrivateKey, error) {
	file, err := j.fs.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file: %w", err)
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file: %w", err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}
	privKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ECDSA private key: %w", err)
	}
	return privKey, nil
}

func (j *JwtAuth) GetPublicKey() *ecdsa.PublicKey {
	return j.publicKey
}

func (j *JwtAuth) GenerateJwt() (string, error) {
	token := jwt.New(jwt.SigningMethodEdDSA)
	claims := token.Claims.(jwt.MapClaims)
	claims["exp"] = time.Now().Add(15 * time.Minute)
	claims["authorized"] = true
	tokenString, err := token.SignedString(j.privateKey)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}
