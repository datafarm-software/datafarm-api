package redis

import (
	"fmt"

	deviceinfo "github.com/datafarm-software/datafarm-api/api/device-info"
	"github.com/redis/go-redis/v9"
)

const TestingDb = 13

func (r *Redis) PrepareDeviceInfo(deviceinfo.Schema) error {
	return nil
}

func (r *Redis) GetQueryFields(deviceId string) (deviceinfo.QueryFields, error) {
	var qf []string
	var err error
	qf, err = r.db.SMembers(ctx, "queryFields:"+deviceId).Result()
	if err != nil {
		err = fmt.Errorf("redis smembers: %v", err)
	}
	return deviceinfo.QueryFields{
		DeviceId:    deviceId,
		QueryFields: append(deviceinfo.GeneralQueryFields, qf...),
	}, err
}

func (r *Redis) GetCompany(deviceId string) (string, error) {
	company, err := r.db.HGet(ctx, "fieldUnit:"+deviceId, "company").Result()
	if err != nil {
		if err == redis.Nil {
			return "", deviceinfo.NotFound
		}
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

func (r *Redis) GetDevices(sr deviceinfo.ScopeRestriction) ([]string, error) {
	var key string
	switch sr.Scope {
	case deviceinfo.DevicesInCompanyInNetwork:
		key = "companyDevices:" + sr.Company
	case deviceinfo.DevicesInNetwork:
		key = "networkIds:" + sr.Network
	case deviceinfo.AllDevices:
		key = "allDevices"
	}
	return r.db.SMembers(ctx, key).Result()
}
