package redis

import (
	"bytes"
	"fmt"
	"os"

	cfy "github.com/geraud22/config-from-yaml"
	"github.com/geraud22/datafarm-api/authoriser"
	mdf "github.com/geraud22/datafarm-api/metadatafetcher"
	"github.com/redis/go-redis/v9"
)

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

type PipelineFunc func(pipe redis.Pipeliner) error

const MaxRetries = 10

func (r *Redis) pipeline(pipeFunc PipelineFunc, keysToWatch ...string) error {
	var watchErr, pipeErr error
RetryLoop:
	for range MaxRetries {
		watchErr = r.db.Watch(ctx, func(tx *redis.Tx) error {
			_, pipeErr = tx.TxPipelined(ctx, pipeFunc)
			return pipeErr
		}, keysToWatch...)
		switch watchErr {
		case nil:
			break RetryLoop
		case redis.TxFailedErr:
			continue
		default:
			return watchErr
		}
	}
	return nil
}

func (t *TestingRedis) PrepareBasicAuth(db map[string]authoriser.UserInfo) error {
	pfn := func(pipe redis.Pipeliner) error {
		for k, v := range db {
			pipe.HSet(ctx, "userInfo:"+k, v)
		}
		return nil
	}
	err := t.redis.pipeline(pfn)
	if err != nil {
		return fmt.Errorf("pipe: %v", err)
	}
	return nil
}

func (t *TestingRedis) PrepareMetadataFetcher(md mdf.Metadata) error {
	return fmt.Errorf("not implemented")
}

func (t *TestingRedis) GetAttachedSensors(deviceId string) ([]string, error) {
	return t.redis.GetAttachedSensors(deviceId)
}
func (t *TestingRedis) GetQueryFields(attachedSensors []string) ([]string, error) {
	return t.redis.GetQueryFields(attachedSensors)
}
func (t *TestingRedis) GetCompany(deviceId string) (string, error) {
	return t.redis.GetCompany(deviceId)
}
func (t *TestingRedis) GetNetwork(deviceId string) (string, error) {
	return t.redis.GetNetwork(deviceId)
}
func (t *TestingRedis) StoreToken(ut authoriser.UserToken) error {
	return t.redis.StoreToken(ut)
}

func (t *TestingRedis) VerifyCredentials(username, passw string) error {
	return t.redis.VerifyCredentials(username, passw)
}
