package redis

import (
	"fmt"

	"github.com/datafarm-software/datafarm-api/api/device"
	"github.com/datafarm-software/datafarm-api/api/device/info"
	"github.com/redis/go-redis/v9"
)

const TestingDb = 13

func (r *Redis) PrepareDeviceInfo(device.Schema) error {
	return nil
}

func (r *Redis) GetQueryFields(deviceId string) (info.QueryFields, error) {
	var qf []string
	var err error
	qf, err = r.db.SMembers(ctx, "queryFields:"+deviceId).Result()
	if err != nil {
		err = fmt.Errorf("redis smembers: %v", err)
	}
	return info.QueryFields{
		DeviceId:    deviceId,
		QueryFields: append(info.GeneralQueryFields, qf...),
	}, err
}

func (r *Redis) GetCompany(deviceId string) (string, error) {
	company, err := r.db.HGet(ctx, "fieldUnit:"+deviceId, "company").Result()
	if err != nil {
		if err == redis.Nil {
			return "", info.NotFound
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

func (r *Redis) GetDevices(sr info.ScopeRestriction) ([]string, error) {
	var key string
	switch sr.Scope {
	case info.DevicesInCompanyInNetwork:
		key = "companyDevices:" + sr.Company
	case info.DevicesInNetwork:
		key = "networkIds:" + sr.Network
	case info.AllDevices:
		key = "allDevices"
	}
	return r.db.SMembers(ctx, key).Result()
}
