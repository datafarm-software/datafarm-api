package redis

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/geraud22/datafarm-api/authoriser"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

var ctx = context.Background()
var once sync.Once

type RedisOpts struct {
	Addr     string `mapstructure:"address" validate:"required"`
	Username string `mapstructure:"username" validate:"required"`
	Password string `mapstructure:"password" validate:"required"`
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

func (r *Redis) CheckCredentials(username, passw string) (authoriser.UserInfo, error) {
	uuid, err := r.db.Get(ctx, "unique:"+username).Result()
	if err != nil {
		return authoriser.UserInfo{}, fmt.Errorf("error getting uuid for username %s: %v", username, err)
	}
	var userInfo authoriser.UserInfo
	if err := r.db.HGetAll(ctx, "user:"+uuid).Scan(&userInfo); err != nil {
		return authoriser.UserInfo{}, fmt.Errorf("error getting password for username %s: %v", username, err)
	}
	userInfo.Network, err = r.db.Get(ctx, "network:"+userInfo.Company).Result()
	if err != nil {
		return authoriser.UserInfo{}, fmt.Errorf("error getting network: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(userInfo.Password), []byte(passw)); err != nil {
		return authoriser.UserInfo{}, fmt.Errorf("passwords did not match: %v", err)
	}
	return authoriser.UserInfo{
		Username: userInfo.Username,
		Company:  userInfo.Company,
		Network:  userInfo.Network,
		Role:     userInfo.Role,
	}, nil
}
