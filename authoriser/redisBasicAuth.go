package authoriser

import (
	"context"
	"fmt"

	"github.com/geraud22/aquahaus-api/api"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

var g_ctx = context.Background()

type redisBasicAuth struct {
	db *redis.Client
}

func ConnectRedis(addr, username, passw string, db int) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Username: username,
		Password: passw,
		DB:       db,
	})
	if _, err := client.Ping(g_ctx).Result(); err != nil {
		return nil, fmt.Errorf("Error connecting to Redis client: %v", err)
	}
	return client, nil
}

func (r *redisBasicAuth) Close() error {
	if err := r.db.Close(); err != nil {
		return fmt.Errorf("error closing redis client: %v", err)
	}
	return nil
}

func (r *redisBasicAuth) CheckCredentials(username, passw string) (api.UserInfo, error) {
	uuid, err := r.db.Get(g_ctx, "unique:"+username).Result()
	if err != nil {
		return api.UserInfo{}, fmt.Errorf("error getting uuid for username %s: %v", username, err)
	}
	userHash, err := r.db.HGetAll(g_ctx, "user:"+uuid).Result()
	if err != nil {
		return api.UserInfo{}, fmt.Errorf("error getting password for username %s: %v", username, err)
	}
	passWordHash := userHash["password"]
	company := userHash["company"]
	// role := userHash["role"]
	network, err := r.db.Get(g_ctx, "network:"+company).Result()
	if err != nil {
		return api.UserInfo{}, fmt.Errorf("error getting network: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(passWordHash), []byte(passw)); err != nil {
		return api.UserInfo{}, fmt.Errorf("passwords did not match: %v", err)
	}
	return api.UserInfo{
		Username: username,
		Company:  company,
		Network:  network,
	}, nil
}
