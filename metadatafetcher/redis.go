package metadatafetcher

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
)

var g_ctx = context.Background()

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

type RedisMetadata struct {
	db *redis.Client
}

func NewRedisMetadata(addr, password string, db int) *RedisMetadata {
	return &RedisMetadata{
		db: connectRedis(addr, password, db),
	}
}

func (r *RedisMetadata) Close() {
	if err := r.db.Close(); err != nil {
		log.Printf("error closing redis client: %v", err)
	}
}

func (r *RedisMetadata) GetCompany(deviceId string) (string, error)                {}
func (r *RedisMetadata) GetNetwork(deviceId string) (string, error)                {}
func (r *RedisMetadata) GetAttachedSensors(deviceId string) ([]string, error)      {}
func (r *RedisMetadata) GetQueryFields(attachedSensors []string) ([]string, error) {}
