package metadatafetcher

import (
	"context"
	"fmt"

	"github.com/geraud22/aquahaus-api/authoriser"
	"github.com/redis/go-redis/v9"
)

var g_ctx = context.Background()

type RedisMetadataOpts struct {
	Addr     string `mapstructure:"address" validate:"required"`
	Username string `mapstructure:"username" validate:"required"`
	Password string `mapstructure:"password" validate:"required"`
	Db       int    `mapstructure:"db" validate:"gte=0"`
}

type RedisMetadata struct {
	db *redis.Client
}

func NewRedisMetadata(opts RedisMetadataOpts) (*RedisMetadata, error) {
	db, err := authoriser.ConnectRedis(opts.Addr, opts.Username, opts.Password, opts.Db)
	if err != nil {
		return nil, err
	}
	return &RedisMetadata{
		db: db,
	}, nil
}

func (r *RedisMetadata) Close() error {
	if err := r.db.Close(); err != nil {
		return fmt.Errorf("error closing redis client: %v", err)
	}
	return nil
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
