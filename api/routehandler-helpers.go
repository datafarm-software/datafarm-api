package api

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/datafarm-software/datafarm-api/api/authstore"
	"github.com/datafarm-software/datafarm-api/api/datafetcher"
	deviceinfo "github.com/datafarm-software/datafarm-api/api/device-info"
	"go.uber.org/zap"
)

func logF(ctx context.Context, fields ...zap.Field) {
	rl, _ := ctx.Value("request-log").(*requestLog)
	if rl != nil {
		rl.fields = append(rl.fields, fields...)
	}
}

const MaxDays = 90
const MaxMinutes = 129600
const MaxSeconds = 7776000
const MaxHours = 2160
const MaxMonths = 3
const LowerCaseO = 0x6f
const Hyphen = 0x2d

func CheckOlderThanNinetyDays(start string) bool {
	if len(start) < 1 {
		return true
	}
	if start[0] != Hyphen {
		return true
	}
	start = strings.ReplaceAll(start, "-", "")
	var suffix string
	if start[len(start)-1] == byte(LowerCaseO) {
		suffix = "mo"
		start = strings.ReplaceAll(start, suffix, "")
	} else {
		suffix = string(start[len(start)-1])
		start = start[:len(start)-1]
	}
	number, err := strconv.Atoi(start)
	if err != nil {
		log.Printf("number conversion error: %v", err)
		return true
	}
	switch suffix {
	case "s":
		if number > MaxSeconds {
			return true
		}
	case "m":
		if number > MaxMinutes {
			return true
		}
	case "h":
		if number > MaxHours {
			return true
		}
	case "d":
		if number > MaxDays {
			return true
		}
	case "mo":
		if number > MaxMonths {
			return true
		}
	default:
		log.Printf("unknown suffix: %v", suffix)
		return true
	}
	return false
}

func formatTimestamp(in *datafetcher.SensorDataRequest) (err error) {
	in.TimeFrame.Start = strings.TrimSpace(in.TimeFrame.Start)
	if RELATIVETIME_REGEX.MatchString(in.TimeFrame.Start) {
		older := CheckOlderThanNinetyDays(in.TimeFrame.Start)
		if older {
			return huma.Error400BadRequest("Relative start time older than 90 days.")
		}
		in.TimeFrame.Stop = ""
	} else {
		rfcStart, err := time.Parse(time.RFC3339Nano, in.TimeFrame.Start)
		if err != nil {
			return huma.Error400BadRequest("Start time is invalid rfc.")
		}
		if rfcStart.UnixMilli() <= time.Now().Add(-90*24*time.Hour).UnixMilli() {
			return huma.Error400BadRequest("Start time is greater than 90 days.")
		}
		if rfcStart.UnixMilli() >= time.Now().UnixMilli() {
			return huma.Error400BadRequest("Start time is in the future.")
		}
		if in.TimeFrame.Stop == "" {
			return huma.Error400BadRequest("No stop time provided.")
		}
		in.TimeFrame.Stop = strings.TrimSpace(in.TimeFrame.Stop)
		rfcStop, err := time.Parse(time.RFC3339Nano, in.TimeFrame.Stop)
		if err != nil {
			return huma.Error400BadRequest("Stop time is invalid rfc.")
		}
		if rfcStart.UnixMilli() >= rfcStop.UnixMilli() {
			return huma.Error400BadRequest("Start time is greater than stop time.")
		}
	}
	return nil
}

func (a *Api) checkAccessToDevice(deviceId string, user authstore.UserInfo) (
	deviceinfo.DeviceInfo, int, error) {
	di := deviceinfo.DeviceInfo{DeviceId: deviceId}
	deviceCompany, err := a.DeviceInfo.GetCompany(deviceId)
	if err != nil {
		if errors.Is(err, deviceinfo.NotFound) {
			return di, http.StatusNotFound, fmt.Errorf(
				"Device not found.")
		}
		return di, http.StatusInternalServerError, err
	}
	if user.Company != deviceCompany {
		if !authstore.HasPermission(authstore.Role(user.Role), authstore.GetAnyCompany) {
			return di, http.StatusUnauthorized, fmt.Errorf("Unauthorized access to this device.")
		}
	}
	deviceNetwork, err := a.DeviceInfo.GetNetwork(deviceId)
	if err != nil {
		if errors.Is(err, deviceinfo.NotFound) {
			return di, http.StatusNotFound, fmt.Errorf(
				"Device not found.")
		}
		return di, http.StatusInternalServerError, err
	}
	if user.Network != deviceNetwork {
		if !authstore.HasPermission(authstore.Role(user.Role), authstore.GetAnyNetwork) {
			return di, http.StatusUnauthorized, fmt.Errorf("Unauthorized access to this device.")
		}
	}
	di.Company = deviceCompany
	di.Network = deviceNetwork
	return di, http.StatusOK, nil
}

func (a *Api) getSensorData(
	ctx context.Context, in *datafetcher.SensorDataRequest) (
	sensorData datafetcher.SensorDataSlice, err error) {
	if err = formatTimestamp(in); err != nil {
		return nil, err
	}
	user, ok := ctx.Value("user").(authstore.UserInfo)
	if !ok {
		logF(ctx, zap.String("domain.error.message",
			"authstore.UserInfo not found in context"))
		return nil, huma.Error500InternalServerError(
			"Internal error getting user.")
	}
	di, code, err := a.checkAccessToDevice(in.Hardware.DeviceId, user)
	if err != nil {
		switch code {
		case http.StatusUnauthorized:
			return nil, huma.Error401Unauthorized(
				"Unauthorized access to this device.")
		case http.StatusNotFound:
			return nil, huma.Error404NotFound(
				"Device Not Found.")
		default:
			logF(ctx, zap.String("deviceinfo.error.message", err.Error()))
			return nil, huma.Error500InternalServerError(
				"Internal error checking acess to DeviceId.")
		}
	}
	di.Start = in.TimeFrame.Start
	di.Stop = in.TimeFrame.Stop
	di.QueryFields = in.Hardware.QueryFields
	if in.Hardware.QueryFields[0] == "all" {
		if !authstore.HasPermission(authstore.Role(user.Role),
			authstore.GetAllQueryFields) {
			return nil, huma.Error401Unauthorized(
				"Unauthorized for all queryfields.")
		}
		qf, err := a.DeviceInfo.GetQueryFields(in.Hardware.DeviceId)
		if err != nil {
			logF(ctx,
				zap.String("deviceinof.error.message",
					fmt.Sprintf(
						"error getting query fields for: %s: %v",
						in.Hardware.DeviceId, err)),
			)
			return nil, huma.Error500InternalServerError(
				"Internal error getting QueryFields for Device.")
		}
		di.QueryFields = qf.QueryFields
	}
	di.Timezone, err = in.Timezone.Location()
	if err != nil {
		return nil, huma.Error400BadRequest(
			"Invalid location. Please try a different IANA Timezone.")
	}
	sensorData, err = a.DataFetcher.GetData(di)
	if err != nil {
		logF(ctx, zap.String("datafetcher.error.message",
			fmt.Sprintf("error getting data: %v", err)),
		)
		return nil, huma.Error500InternalServerError(
			"Internal error fetching data.")
	}
	return sensorData, nil
}
