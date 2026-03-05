package redis

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"slices"

	cfy "github.com/geraud22/config-from-yaml"
	"github.com/geraud22/datafarm-api/authoriser"
	mdf "github.com/geraud22/datafarm-api/metadatafetcher"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
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
	var hashedPassword []byte
	var err error
	pfn := func(pipe redis.Pipeliner) error {
		for k, v := range db {
			hashedPassword, err = bcrypt.GenerateFromPassword(
				[]byte(v.Password), bcrypt.DefaultCost)
			if err != nil {
				break
			}
			v.Password = string(hashedPassword)
			pipe.Set(ctx, "unique:"+k, k, 0)
			pipe.HSet(ctx, "user:"+k, v)
		}
		return nil
	}
	err = t.redis.pipeline(pfn)
	if err != nil {
		return fmt.Errorf("pipe: %v", err)
	}
	return nil
}

func (t *TestingRedis) PrepareMetadataFetcher(schema mdf.Schema) error {
	pfn := func(pipe redis.Pipeliner) error {
		for _, d := range schema.DeviceCompanies {
			pipe.SAdd(ctx, "deviceIds", d.DeviceId)
			pipe.HSet(ctx, "fieldUnit:"+d.DeviceId, "company", d.Company)
		}
		for _, d := range schema.DeviceNetworks {
			pipe.SAdd(ctx, "deviceIds", d.DeviceId)
			pipe.HSet(ctx, "fieldUnit:"+d.DeviceId, "network", d.Network)
		}
		for _, d := range schema.DeviceToSensors {
			pipe.SAdd(ctx, "deviceIds", d.DeviceId)
			pipe.SAdd(ctx, "attachedSensors:"+d.DeviceId, d.AttachedSensors)
		}
		for _, d := range schema.SensorToQF {
			pipe.SAdd(ctx, "queryFields:"+d.Sensor, d.QueryFields)
		}
		for _, u := range schema.UserTokens {
			pipe.SAdd(ctx, "usersWithToken", u.Username)
			pipe.Set(ctx, "userToken:"+u.Username, u.Token, 0)
			pipe.Set(ctx, "tokenUser:"+u.Token, u.Username, 0)
		}
		return nil
	}
	if err := t.redis.pipeline(pfn); err != nil {
		return err
	}
	return nil
}

func (t *TestingRedis) GetSnapshot() *mdf.Schema {
	schema := &mdf.Schema{}
	deviceIds, err := t.redis.db.SMembers(ctx, "deviceIds").Result()
	if err != nil {
		log.Printf("getting deviceIds: %v", err)
		return schema
	}
	cmdVec := make(map[string]map[string]any)
	pfn := func(pipe redis.Pipeliner) error {
		for _, id := range deviceIds {
			cmdVec[id]["company"] = pipe.HGet(ctx, "fieldUnit:"+id, "company")
			cmdVec[id]["network"] = pipe.HGet(ctx, "fieldUnit:"+id, "network")
			cmdVec[id]["attachedSensors"] = pipe.SMembers(ctx, "attachedSensors:"+id)
		}
		return nil
	}
	if err := t.redis.pipeline(pfn); err != nil {
		log.Printf("pipeline: %v", err)
		return schema
	}
	var company, network string
	var deviceSensors []string
	var uniqueSensors []string
	for _, id := range deviceIds {
		company = getStringCmd(cmdVec[id]["company"])
		schema.DeviceCompanies = append(schema.DeviceCompanies,
			mdf.DeviceToCompany{DeviceId: id, Company: company})
		network = getStringCmd(cmdVec[id]["network"])
		schema.DeviceNetworks = append(schema.DeviceNetworks,
			mdf.DeviceToNetwork{DeviceId: id, Network: network})
		deviceSensors = getStringSliceCmd(cmdVec[id]["attachedSensors"])
		schema.DeviceToSensors = append(schema.DeviceToSensors,
			mdf.DeviceToSensor{DeviceId: id, AttachedSensors: deviceSensors})
		for _, s := range deviceSensors {
			if !slices.Contains(uniqueSensors, s) {
				uniqueSensors = append(uniqueSensors, s)
			}
		}
	}

	var qf []string
	for _, s := range uniqueSensors {
		qf, _ = t.redis.db.SMembers(ctx, "attachedSensors:"+s).Result()
		schema.SensorToQF = append(schema.SensorToQF,
			mdf.SensorToQueryFields{Sensor: s, QueryFields: qf})
	}

	usersWithToken, _ := t.redis.db.SMembers(ctx, "usersWithToken").Result()
	for _, u := range usersWithToken {
		token, _ := t.redis.db.Get(ctx, "userToken:"+u).Result()
		schema.UserTokens = append(schema.UserTokens,
			mdf.UserToken{Username: u, Token: token})
	}
	return schema
}

func getStringCmd(cmd any) string {
	c, ok := cmd.(*redis.StringCmd)
	if !ok {
		log.Printf("not a string cmd")
		return ""
	}
	company, _ := c.Result()
	return company
}

func getStringSliceCmd(cmd any) []string {
	c, ok := cmd.(*redis.StringSliceCmd)
	if !ok {
		log.Printf("not a string slice cmd")
		return nil
	}
	slice, _ := c.Result()
	return slice
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

func (t *TestingRedis) GetUser(token string) (string, error) {
	return t.redis.GetUser(token)
}

func (t *TestingRedis) StoreToken(ut authoriser.UserToken) error {
	t.redis.db.SAdd(ctx, "usersWithToken", ut.Username)
	return t.redis.StoreToken(ut)
}

func (t *TestingRedis) DeleteToken(tr authoriser.TokenResponse) error {
	username, _ := t.redis.db.Get(ctx, "tokenUser:"+tr.Token).Result()
	t.redis.db.SRem(ctx, "usersWithToken", username)
	return t.redis.DeleteToken(tr)
}

func (t *TestingRedis) VerifyCredentials(username, passw string) error {
	return t.redis.VerifyCredentials(username, passw)
}
