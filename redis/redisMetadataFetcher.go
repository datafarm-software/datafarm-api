package redis

import (
	"fmt"

	"github.com/geraud22/datafarm-api/authoriser"
	mdf "github.com/geraud22/datafarm-api/metadatafetcher"
)

const TestingDb = 13

func (r *Redis) GetSnapshot() *mdf.Schema {
	return nil
}

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

func (r *Redis) StoreToken(ut authoriser.UserToken) error {
	err := r.db.Set(ctx, "userToken:"+ut.Username, ut.Token, ut.Expiration).Err()
	if err != nil {
		return err
	}
	err = r.db.Set(ctx, "tokenUser:"+ut.Token, ut.Username, ut.Expiration).Err()
	if err != nil {
		return err
	}
	return nil
}

func (r *Redis) DeleteToken(tr authoriser.TokenResponse) error {
	username, err := r.db.Get(ctx, "tokenUser:"+tr.Token).Result()
	if err != nil {
		return err
	}
	err = r.db.Del(ctx, "userToken:"+username).Err()
	if err != nil {
		return err
	}
	err = r.db.Del(ctx, "tokenUser:"+tr.Token).Err()
	if err != nil {
		return err
	}
	return nil
}

func (r *Redis) GetUser(token string) (string, error) {
	return r.db.Get(ctx, "tokenUser:"+token).Result()
}
