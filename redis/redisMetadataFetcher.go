package redis

import (
	"fmt"
)

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
