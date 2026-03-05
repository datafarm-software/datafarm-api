package redis

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/geraud22/datafarm-api/authstore"
	"github.com/geraud22/datafarm-api/tokenprovider"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

var ctx = context.Background()
var once sync.Once

type RedisOpts struct {
	Addr     string `mapstructure:"address" validate:"required"`
	Username string `mapstructure:"username" validate:"omitempty,alphanum"`
	Password string `mapstructure:"password" validate:"omitempty"`
	Db       int    `mapstructure:"db" validate:"gte=0"`
}

type Redis struct {
	db *redis.Client
}

func NewRedis(opts RedisOpts) (*Redis, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     opts.Addr,
		Username: opts.Username,
		Password: opts.Password,
		DB:       opts.Db,
	})
	if _, err := client.Ping(ctx).Result(); err != nil {
		return nil, fmt.Errorf("Error connecting to Redis client: %v", err)
	}
	return &Redis{
		db: client,
	}, nil
}

func (r *Redis) Close() error {
	once.Do(
		func() {
			if err := r.db.Close(); err != nil {
				log.Printf("error closing redis client: %v", err)
			}
		})
	return nil
}

func (r *Redis) PrepareBasicAuth(map[string]authstore.UserInfo) error {
	return nil
}

func (r *Redis) GetActiveTokens() []authstore.UserInfo {
	return nil
}

func (r *Redis) VerifyCredentials(username, passw string) error {
	uuid, err := r.db.Get(ctx, "unique:"+username).Result()
	if err != nil {
		return fmt.Errorf("error getting uuid for username %s: %v", username, err)
	}
	passwordHash, err := r.db.HGet(ctx, "user:"+uuid, "password").Result()
	if err != nil {
		return fmt.Errorf("error getting password for username %s: %v", username, err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(passw)); err != nil {
		return fmt.Errorf("passwords did not match: %v", err)
	}
	return nil
}

func (r *Redis) StoreToken(ut authstore.UserToken) error {
	err := r.db.Set(ctx, "userToken:"+ut.Username, ut.Token, ut.Expiration).Err()
	if err != nil {
		return err
	}
	err = r.db.Set(ctx, "tokenUser:"+ut.Token, ut.Username, ut.Expiration).Err()
	if err != nil {
		return err
	}
	return nil
}

func (r *Redis) DeleteToken(tr tokenprovider.TokenResponse) error {
	username, err := r.db.Get(ctx, "tokenUser:"+tr.Token).Result()
	if err != nil {
		return err
	}
	err = r.db.Del(ctx, "userToken:"+username).Err()
	if err != nil {
		return err
	}
	err = r.db.Del(ctx, "tokenUser:"+tr.Token).Err()
	if err != nil {
		return err
	}
	return nil
}

func (r *Redis) GetUser(token string) (authstore.UserInfo, error) {
	return r.db.Get(ctx, "tokenUser:"+token).Result()
}
