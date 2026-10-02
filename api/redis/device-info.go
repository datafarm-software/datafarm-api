package redis

import (
	"fmt"
	"strings"

	"github.com/datafarm-software/datafarm-api/api/sensor"
	"github.com/datafarm-software/datafarm-api/api/sensor/info"
	"github.com/redis/go-redis/v9"
)

const TestingDb = 13

func (r *Redis) PrepareDeviceInfo(sensor.Schema) error {
	return nil
}

func (r *Redis) GetQueryFields(deviceId sensor.DeviceId) (info.QueryFields, error) {
	var qf []string
	var err error
	qf, err = r.db.SMembers(ctx, "queryFields:"+string(deviceId)).Result()
	if err != nil {
		if strings.Contains(err.Error(), "connect:") {
			return info.QueryFields{}, sensor.NoConnection
		}
		if err == redis.Nil {
			return info.QueryFields{}, sensor.NotFound
		}
		err = fmt.Errorf("redis smembers: %v", err)
	}
	if len(qf) < 1 {
		err = sensor.NotFound
	}
	return info.QueryFields{
		DeviceId:    deviceId,
		QueryFields: append(info.GeneralQueryFields, qf...),
	}, err
}

func (r *Redis) GetCompany(deviceId sensor.DeviceId) (string, error) {
	company, err := r.db.HGet(ctx, "fieldUnit:"+string(deviceId), "company").Result()
	if err != nil {
		if strings.Contains(err.Error(), "connect:") {
			return "", sensor.NoConnection
		}
		if err == redis.Nil {
			return "", sensor.NotFound
		}
	}
	return company, err
}

func (r *Redis) GetNetwork(deviceId sensor.DeviceId) (string, error) {
	network, err := r.db.HGet(ctx, "fieldUnit:"+string(deviceId), "network").Result()
	if err != nil {
		if strings.Contains(err.Error(), "connect:") {
			return "", sensor.NoConnection
		}
		if err == redis.Nil {
			return "", sensor.NotFound
		}
	}
	return network, err
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
	sc, err := r.db.SMembers(ctx, key).Result()
	if err != nil {
		if strings.Contains(err.Error(), "connect:") {
			return nil, sensor.NoConnection
		}
	}
	return sc, nil
}
