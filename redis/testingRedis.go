package redis

import (
	"bytes"
	"fmt"
	"log"
	"os"

	"github.com/datafarm-software/datafarm-api/authstore"
	deviceinfo "github.com/datafarm-software/datafarm-api/device-info"
	cfy "github.com/geraud22/config-from-yaml"
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
	})
	if topErr != nil {
		return nil, topErr
	}
	if addr != "" {
		testingRedisOpts.Addr = addr
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

func (t *TestingRedis) PrepareAuthStore(mockDb authstore.Schema) error {
	var hashedPassword []byte
	var err error
	pfn := func(pipe redis.Pipeliner) error {
		for _, u := range mockDb.UserTokens {
			pipe.SAdd(ctx, "usersWithToken", u.Username)
			pipe.Set(ctx, "userToken:"+u.Username, u.Token, u.Expiration)
			pipe.Set(ctx, "tokenUser:"+u.Token, u.Username, u.Expiration)
		}
		for _, user := range mockDb.UserInfo {
			hashedPassword, err = bcrypt.GenerateFromPassword(
				[]byte(user.Password), bcrypt.DefaultCost)
			if err != nil {
				return err
			}
			user.Password = string(hashedPassword)
			pipe.Set(ctx, "unique:"+user.Username, user.Username, 0)
			pipe.HSet(ctx, "user:"+user.Username, user)
		}
		return nil
	}
	err = t.redis.pipeline(pfn)
	if err != nil {
		return fmt.Errorf("pipe: %v", err)
	}
	return nil
}

func (t *TestingRedis) PrepareDeviceInfo(schema deviceinfo.Schema) error {
	pfn := func(pipe redis.Pipeliner) error {
		for _, d := range schema.DeviceCompanies {
			pipe.SAdd(ctx, "allDevices", d.DeviceId)
			pipe.SAdd(ctx, "companyDevices:"+d.Company, d.DeviceId)
			pipe.SAdd(ctx, "deviceIds", d.DeviceId)
			pipe.HSet(ctx, "fieldUnit:"+d.DeviceId, "company", d.Company)
		}
		for _, d := range schema.DeviceNetworks {
			pipe.SAdd(ctx, "allDevices", d.DeviceId)
			pipe.SAdd(ctx, "networkIds:"+d.Network, d.DeviceId)
			pipe.SAdd(ctx, "deviceIds", d.DeviceId)
			pipe.HSet(ctx, "fieldUnit:"+d.DeviceId, "network", d.Network)
		}
		for _, d := range schema.DeviceToQF {
			pipe.SAdd(ctx, "allDevices", d.DeviceId)
			pipe.SAdd(ctx, "deviceIds", d.DeviceId)
			pipe.SAdd(ctx, "queryFields:"+d.DeviceId, d.QueryFields)
		}
		return nil
	}
	if err := t.redis.pipeline(pfn); err != nil {
		return err
	}
	return nil
}

func (t *TestingRedis) GetSnapshot() *deviceinfo.Schema {
	schema := &deviceinfo.Schema{}
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
			cmdVec[id]["queryFields"] = pipe.SMembers(ctx, "queryFields:"+id)
		}
		return nil
	}
	if err := t.redis.pipeline(pfn); err != nil {
		log.Printf("pipeline: %v", err)
		return schema
	}
	var company, network string
	var queryFields []string
	for _, id := range deviceIds {
		company = getStringCmd(cmdVec[id]["company"])
		schema.DeviceCompanies = append(schema.DeviceCompanies,
			deviceinfo.DeviceToCompany{DeviceId: id, Company: company})
		network = getStringCmd(cmdVec[id]["network"])
		schema.DeviceNetworks = append(schema.DeviceNetworks,
			deviceinfo.DeviceToNetwork{DeviceId: id, Network: network})
		queryFields = getStringSliceCmd(cmdVec[id]["queryFields"])
		schema.DeviceToQF = append(schema.DeviceToQF,
			deviceinfo.DeviceToQueryFields{DeviceId: id, QueryFields: queryFields})
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

func (t *TestingRedis) GetQueryFields(deviceId string) (deviceinfo.QueryFields, error) {
	return t.redis.GetQueryFields(deviceId)
}

func (t *TestingRedis) GetCompany(deviceId string) (string, error) {
	return t.redis.GetCompany(deviceId)
}

func (t *TestingRedis) GetNetwork(deviceId string) (string, error) {
	return t.redis.GetNetwork(deviceId)
}

func (t *TestingRedis) GetUser(token string) (authstore.UserInfo, error) {
	return t.redis.GetUser(token)
}

func (t *TestingRedis) StoreToken(ut authstore.UserToken) error {
	t.redis.db.SAdd(ctx, "usersWithToken", ut.Username)
	return t.redis.StoreToken(ut)
}

func (t *TestingRedis) DeleteToken(ut authstore.UserToken) error {
	username, _ := t.redis.db.Get(ctx, "tokenUser:"+ut.Token).Result()
	t.redis.db.SRem(ctx, "usersWithToken", username)
	return t.redis.DeleteToken(ut)
}

func (t *TestingRedis) VerifyCredentials(username, passw string) error {
	return t.redis.VerifyCredentials(username, passw)
}

func (t *TestingRedis) GetActiveTokens() []authstore.UserToken {
	usersWithToken, _ := t.redis.db.SMembers(ctx, "usersWithToken").Result()
	userTokens := make([]authstore.UserToken, 0, len(usersWithToken))
	for _, u := range usersWithToken {
		ttl, _ := t.redis.db.TTL(ctx, "userToken:"+u).Result()
		token, _ := t.redis.db.Get(ctx, "userToken:"+u).Result()
		userTokens = append(userTokens, authstore.UserToken{
			Username: u, Token: token, Expiration: ttl})
	}
	return userTokens
}

func (t *TestingRedis) GetDevices(sr deviceinfo.ScopeRestriction) ([]string, error) {
	return t.redis.GetDevices(sr)
}

func (t *TestingRedis) GetToken(username string) (authstore.UserToken, error) {
	return t.redis.GetToken(username)
}
