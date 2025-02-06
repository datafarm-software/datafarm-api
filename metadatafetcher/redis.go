package metadatafetcher

import (
	"context"
	"fmt"

	"github.com/geraud22/aquahaus-api/authoriser"
	"github.com/redis/go-redis/v9"
)

var g_ctx = context.Background()

type RedisMetadata struct {
	db *redis.Client
}

func NewRedisMetadata(addr, password string, db int) *RedisMetadata {
	return &RedisMetadata{
		db: authoriser.ConnectRedis(addr, password, db),
	}
}

func (r *RedisMetadata) Close() error {
	if err := r.db.Close(); err != nil {
		return fmt.Errorf("error closing redis client: %v", err)
	}
	return nil
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
	key := fmt.Sprintf("attachedSensors:%s", deviceId)
	attachedSensors, err := r.db.SMembers(g_ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("redis: %v", err)
	}
	return attachedSensors, nil
}

func (r *RedisMetadata) GetQueryFields(attachedSensors []string) ([]string, error) {
	queryfields := make([]string, 0)
	for _, a := range attachedSensors {
		key := fmt.Sprintf("queryFields:%s", a)
		qf, err := r.db.SMembers(g_ctx, key).Result()
		if err != nil {
			return nil, fmt.Errorf("redis smembers: %v", err)
		}
		queryfields = append(queryfields, qf...)
	}
	return queryfields, nil
}
