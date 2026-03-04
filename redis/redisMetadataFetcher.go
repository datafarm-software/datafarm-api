package redis

import (
	"bytes"
	"fmt"
	"os"

	cfy "github.com/geraud22/config-from-yaml"
	"github.com/geraud22/datafarm-api/authoriser"
	"github.com/redis/go-redis/v9"
)

const TestingDb = 13

func (r *Redis) GetAttachedSensors(deviceId string) ([]string, error) {
	key := fmt.Sprintf("attachedSensors:%s", deviceId)
	attachedSensors, err := r.db.SMembers(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("redis: %v", err)
	}
	return attachedSensors, nil
}

func (r *Redis) GetQueryFields(attachedSensors []string) ([]string, error) {
	queryfields := make([]string, 0)
	for _, a := range attachedSensors {
		key := fmt.Sprintf("queryFields:%s", a)
		qf, err := r.db.SMembers(ctx, key).Result()
		if err != nil {
			return nil, fmt.Errorf("redis smembers: %v", err)
		}
		queryfields = append(queryfields, qf...)
	}
	return queryfields, nil
}

func (r *Redis) GetCompany(deviceId string) (string, error) {
	company, err := r.db.HGet(ctx, "fieldUnit:"+deviceId, "company").Result()
	if err != nil {
		return "", err
	}
	return company, nil
}

func (r *Redis) GetNetwork(deviceId string) (string, error) {
	network, err := r.db.HGet(ctx, "fieldUnit:"+deviceId, "network").Result()
	if err != nil {
		return "", err
	}
	return network, nil
}

var testingRedisOpts RedisOpts

type TestingRedis struct {
	redis *Redis
}

func NewTestingRedis(addr string) (*TestingRedis, error) {
	var topErr error
	once.Do(func() {
		config, err := os.ReadFile("../config.yml")
		if err != nil {
			topErr = err
			return
		}
		opts, err := cfy.LoadConfig[struct {
			RedisOpts RedisOpts `mapstructure:"redis"`
		}](bytes.NewReader(config), "yaml", nil)
		if err != nil {
			topErr = err
			return
		}
		testingRedisOpts = opts.RedisOpts
		if addr != "" {
			testingRedisOpts.Addr = addr
		}
	})
	if topErr != nil {
		return nil, topErr
	}
	testingRedisOpts.Db = TestingDb
	db, err := NewRedis(testingRedisOpts)
	if err != nil {
		return nil, err
	}
	return &TestingRedis{
		redis: db,
	}, nil
}

func (t *TestingRedis) Close() error {
	err := t.redis.db.FlushDB(ctx).Err()
	if err != nil && err != redis.Nil {
		return nil
	}
	return t.redis.Close()
}

func (t *TestingRedis) PrepareDb(db map[string]authoriser.UserInfo) error {
	return fmt.Errorf("not implemented")
}
