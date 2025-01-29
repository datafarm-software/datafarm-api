package metadatafetcher

import (
	"context"
	"fmt"
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

func (r *RedisMetadata) GetMapValue(deviceId, mapKey string) (string, error) {
	hashName := fmt.Sprintf("fieldUnit:%s", deviceId)
	company, err := r.db.HGet(g_ctx, hashName, mapKey).Result()
	if err != nil {
		return "", fmt.Errorf("redis: %v", err)
	}
	return company, nil
}
func (r *RedisMetadata) GetAttachedSensors(deviceId string) ([]string, error) {
	key := fmt.Sprintf("fieldUnit:%s:attached_sensors", deviceId)
	attachedSensors, err := r.db.LRange(g_ctx, key, 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("redis: %v", err)
	}
	return attachedSensors, nil
}
func (r *RedisMetadata) GetQueryFields(attachedSensors []string) ([]string, error) {
	queryfields := make([]string, 0)
	for _, a := range attachedSensors {
		key := fmt.Sprintf("sensorType:%s:query_fields", a)
		qf, err := r.db.SMembers(g_ctx, key).Result()
		if err != nil {
			return nil, fmt.Errorf("redis smembers: %v", err)
		}
		queryfields = append(queryfields, qf...)
	}
	return queryfields, nil
}
