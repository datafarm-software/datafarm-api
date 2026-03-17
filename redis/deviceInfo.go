package redis

import (
	"fmt"

	deviceinfo "github.com/datafarm-software/datafarm-api/device-info"
)

const TestingDb = 13

func (r *Redis) PrepareDeviceInfo(deviceinfo.Schema) error {
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
