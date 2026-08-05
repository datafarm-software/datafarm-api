package tokenprovider

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"io/fs"
	"log"
	"time"

	"github.com/datafarm-software/datafarm-api/api/authstore"
	"github.com/golang-jwt/jwt/v5"
)

const THREE_HOURS = 3 * time.Hour
const TWELVE_HOURS = 12 * time.Hour
const DemoViewer = "4"

type jwtAuth struct {
	publicKey  *ecdsa.PublicKey
	privateKey *ecdsa.PrivateKey
}

func NewJwtAuth(fs fs.FS, privateKeyPath, publicKeyPath string) (*jwtAuth, error) {
	if fs == nil || publicKeyPath == "" {
		return nil, fmt.Errorf("not all options present")
	}
	var jwtAuth jwtAuth
	publicKey, err := jwtAuth.loadECDSAPublicKey(fs, publicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("error loading public key: %v", err)
	}
	jwtAuth.publicKey = publicKey
	privateKey, err := jwtAuth.loadECDSAPrivateKey(fs, privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("error loading public key: %v", err)
	}
	jwtAuth.privateKey = privateKey
	return &jwtAuth, nil
}

func (j *jwtAuth) Close() error {
	return nil
}

func (j *jwtAuth) loadECDSAPublicKey(fs fs.FS, filePath string) (*ecdsa.PublicKey, error) {
	file, err := fs.Open(filePath)
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

func (j *jwtAuth) loadECDSAPrivateKey(fs fs.FS, filePath string) (*ecdsa.PrivateKey, error) {
	file, err := fs.Open(filePath)
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

func (j *jwtAuth) getPublicKey() *ecdsa.PublicKey {
	return j.publicKey
}

func (j *jwtAuth) GenerateToken(username string) (authstore.UserToken, error) {
	token := jwt.New(jwt.SigningMethodES256)
	claims := token.Claims.(jwt.MapClaims)
	exp := time.Now().UTC().Add(THREE_HOURS)
	claims["exp"] = jwt.NewNumericDate(exp)
	claims["authorized"] = true
	tokenString, err := token.SignedString(j.privateKey)
	if err != nil {
		return authstore.UserToken{}, err
	}
	return authstore.UserToken{
		Username:   username,
		Expiration: THREE_HOURS,
		Token:      tokenString,
	}, nil
}

func (j *jwtAuth) IsValidToken(tr LoginResponse) bool {
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tr.Body, claims,
		func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodECDSA); !ok {
				return nil, nil
			}
			return j.getPublicKey(), nil
		})
	if err != nil {
		log.Printf("token parsing error: %v", err)
		return false
	}
	if !token.Valid {
		log.Println("Invalid token provided")
		return false
	}
	return true
}
