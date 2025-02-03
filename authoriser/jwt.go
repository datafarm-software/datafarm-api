package authoriser

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"io/fs"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

var g_ctx = context.Background()

type Authoriser struct {
	jwtAuth   *JwtAuth
	basicAuth *RedisBasicAuth
}

type RedisBasicAuth struct {
	db *redis.Client
}

func NewAuthoriser(addr, passw, privateKeyPath, publicKeyPath string, db int, fs fs.FS) (*Authoriser, error) {
	jwtAuth, err := NewJwtAuth(fs, privateKeyPath, publicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("jwt auth error: %v", err)
	}
	return &Authoriser{
		jwtAuth:   jwtAuth,
		basicAuth: NewRedisBasicAuth(addr, passw, db),
	}, nil
}

func connectRedis(addr, passw string, db int) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: passw,
		DB:       db,
	})
	if _, err := client.Ping(g_ctx).Result(); err != nil {
		log.Fatalf("Error connecting to Redis client: %v", err)
	}
	return client
}

func NewRedisBasicAuth(addr, password string, db int) *RedisBasicAuth {
	return &RedisBasicAuth{
		db: connectRedis(addr, password, db),
	}
}

func (r *RedisBasicAuth) Close() error {
	if err := r.db.Close(); err != nil {
		return fmt.Errorf("error closing redis client: %v", err)
	}
	return nil
}

type JwtAuth struct {
	publicKey  *ecdsa.PublicKey
	privateKey *ecdsa.PrivateKey
}

func NewJwtAuth(fs fs.FS, privateKeyPath, publicKeyPath string) (*JwtAuth, error) {
	if fs == nil || publicKeyPath == "" {
		return nil, fmt.Errorf("not all options present")
	}
	var jwtAuth JwtAuth
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

func (j *JwtAuth) loadECDSAPublicKey(fs fs.FS, filePath string) (*ecdsa.PublicKey, error) {
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

func (j *JwtAuth) loadECDSAPrivateKey(fs fs.FS, filePath string) (*ecdsa.PrivateKey, error) {
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

func (a *Authoriser) GetPublicKey() *ecdsa.PublicKey {
	return a.jwtAuth.publicKey
}

func (a *Authoriser) GenerateJwt() (string, error) {
	token := jwt.New(jwt.SigningMethodES256)
	claims := token.Claims.(jwt.MapClaims)
	claims["exp"] = jwt.NewNumericDate(time.Now().UTC().Add(15 * time.Minute))
	claims["authorized"] = true
	tokenString, err := token.SignedString(a.jwtAuth.privateKey)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}
