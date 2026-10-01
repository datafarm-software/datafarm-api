package api

import (
	"context"
	"errors"
	"fmt"
	stdlog "log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/datafarm-software/datafarm-api/api/authstore"
	"github.com/datafarm-software/datafarm-api/api/sensor"
	"github.com/datafarm-software/datafarm-api/api/sensor/data"
	"github.com/datafarm-software/datafarm-api/api/sensor/info"
	"github.com/datafarm-software/telemetry/logging"
)

func (a *Api) httpErr(humaCtx huma.Context, w http.ResponseWriter, msg string, code int) {
	span, _ := a.Tracer.SpanFromContext(humaCtx.Context())
	if span.IsValid() {
		traceId := span.TraceId()
		if traceId != "" {
			msg += fmt.Sprintf(" Request TraceID: %s", traceId)
		}
	}
	humaCtx.SetStatus(code)
	w.Write([]byte(msg))
}

func logFromTag(ctx context.Context, a any) error {
	m, err := logging.FromTagMetadata(a)
	if err != nil {
		return err
	}
	log, ok := ctx.Value("request-log").(logging.LogAccumulator)
	if !ok {
		return huma.Error500InternalServerError(
			"Internal error while getting request log.")
	}
	log.AddMetadata(m)
	return nil
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
		stdlog.Printf("number conversion error: %v", err)
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
		stdlog.Printf("unknown suffix: %v", suffix)
		return true
	}
	return false
}

func formatTimestamp(in *data.SensorDataRequest) (err error) {
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

func (a *Api) checkAccess(log logging.LogAccumulator, user authstore.UserInfo, deviceId string) (
	di sensor.Device, err error) {
	di = sensor.Device{DeviceId: deviceId}
	deviceCompany, err := a.DeviceInfo.GetCompany(deviceId)
	if err != nil {
		if errors.Is(err, info.NotFound) {
			return di, huma.Error404NotFound("Device Not Found.")
		}
		log.AddMetadata(logging.Metadata{
			"source":        {"checkAccess.deviceInfo.getCompany"},
			"error.message": {err.Error()}})
		return di, huma.Error500InternalServerError(
			"Internal error checking access to DeviceId.")
	}
	if user.Company != deviceCompany {
		if !authstore.HasPermission(authstore.Role(user.Role), authstore.GetAnyCompany) {
			return di, huma.Error401Unauthorized(
				"Unauthorized access to this sensor.")
		}
	}
	deviceNetwork, err := a.DeviceInfo.GetNetwork(deviceId)
	if err != nil {
		if errors.Is(err, info.NotFound) {
			return di, huma.Error404NotFound("Device Not Found.")
		}
		log.AddMetadata(logging.Metadata{
			"source":        {"checkAccess.deviceInfo.getNetwork"},
			"error.message": {err.Error()}})
		return di, huma.Error500InternalServerError(
			"Internal error checking access to DeviceId.")
	}
	if user.Network != deviceNetwork {
		if !authstore.HasPermission(authstore.Role(user.Role), authstore.GetAnyNetwork) {
			return di, huma.Error401Unauthorized("Unauthorized access to this sensor.")
		}
	}
	di.Company = deviceCompany
	di.Network = deviceNetwork
	return
}

func (a *Api) getSensorData(
	ctx context.Context, in *data.SensorDataRequest) (
	sensorData data.SensorDataSlice, err error) {
	if err = formatTimestamp(in); err != nil {
		return nil, err
	}
	di, err := a.deviceInfoIfAccessAndPermission(ctx, in.DeviceId, authstore.GetSensorData)
	if err != nil {
		return nil, err
	}
	di.Start = in.TimeFrame.Start
	di.Stop = in.TimeFrame.Stop
	di.QueryFields = in.Hardware.QueryFields
	log, ok := ctx.Value("request-log").(logging.LogAccumulator)
	if !ok {
		return nil, huma.Error500InternalServerError(
			"Internal error while getting request log.")
	}
	if in.Hardware.QueryFields[0] == "all" {
		user, _ := a.user(ctx)
		if !authstore.HasPermission(authstore.Role(user.Role),
			authstore.GetAllQueryFields) {
			return nil, huma.Error401Unauthorized(
				"Unauthorized for all QueryFields.")
		}
		qf, err := a.DeviceInfo.GetQueryFields(in.Hardware.DeviceId)
		if err != nil {
			log.AddMetadata(logging.Metadata{
				"source":        {"getSensorData.deviceInfo.getQueryFields"},
				"error.message": {err.Error()}})
			return nil, huma.Error500InternalServerError(
				"Internal error getting QueryFields.")
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
		if errors.Is(err, sensor.NoData) ||
			errors.Is(err, sensor.NoConnection) {
			return sensorData, err
		}
		log.AddMetadata(logging.Metadata{
			"source":        {"getSensorData.dataFetcher.getData"},
			"error.message": {err.Error()}})

		return nil, huma.Error500InternalServerError(
			"Internal error getting SensorData.")
	}
	return sensorData, nil
}

func (a *Api) getLatestSensorData(
	ctx context.Context, in *data.LatestSensorDataRequest) (
	sd data.SensorData, err error) {
	log, ok := ctx.Value("request-log").(logging.LogAccumulator)
	if !ok {
		return sd, huma.Error500InternalServerError(
			"Internal error while getting request log.")
	}
	di, err := a.deviceInfoIfAccessAndPermission(ctx, in.DeviceId, authstore.GetSensorData)
	if err != nil {
		return sd, err
	}
	di.QueryFields = in.Hardware.QueryFields
	if in.Hardware.QueryFields[0] == "all" {
		user, _ := a.user(ctx)
		if !authstore.HasPermission(authstore.Role(user.Role),
			authstore.GetAllQueryFields) {
			return sd, huma.Error401Unauthorized(
				"Unauthorized for all QueryFields.")
		}
		qf, err := a.DeviceInfo.GetQueryFields(in.Hardware.DeviceId)
		if err != nil {
			log.AddMetadata(logging.Metadata{
				"source":        {"getSensorData.deviceInfo.getQueryFields"},
				"error.message": {err.Error()}})
			return sd, huma.Error500InternalServerError(
				"Internal error getting QueryFields.")
		}
		di.QueryFields = qf.QueryFields
	}
	di.Timezone, err = in.Timezone.Location()
	if err != nil {
		return sd, huma.Error400BadRequest(
			"Invalid location. Please try a different IANA Timezone.")
	}
	sd, err = a.DataFetcher.GetLatestData(di)
	if err != nil {
		if errors.Is(err, sensor.NoData) ||
			errors.Is(err, sensor.NoConnection) {
			return sd, err
		}
		log.AddMetadata(logging.Metadata{
			"source":        {"getLatestSensorData.dataFetcher.getLatestData"},
			"error.message": {err.Error()}})
		return sd, huma.Error500InternalServerError(
			"Internal error getting Latest SensorData.")
	}
	return
}

func (a *Api) getQueryFields(ctx context.Context, in *info.QueryFieldsRequest) (
	qf info.QueryFields, err error) {
	_, err = a.deviceInfoIfAccessAndPermission(ctx, in.DeviceId, authstore.GetAllQueryFields)
	if err != nil {
		return
	}
	log, ok := ctx.Value("request-log").(logging.LogAccumulator)
	if !ok {
		return qf, huma.Error500InternalServerError(
			"Internal error while getting request log.")
	}
	qf, err = a.DeviceInfo.GetQueryFields(in.DeviceId)
	if err != nil {
		log.AddMetadata(logging.Metadata{
			"source":        {"getQueryFields.deviceInfo.getQueryFields"},
			"error.message": {err.Error()}})
		return qf, huma.Error500InternalServerError(
			"Internal error while getting QueryFields.")
	}
	return
}

func (a *Api) deviceInfoIfAccessAndPermission(ctx context.Context,
	deviceId string, permission authstore.Permission) (
	di sensor.Device, err error) {
	user, ok := ctx.Value("user").(authstore.UserInfo)
	if !ok {
		return di, huma.Error500InternalServerError(
			"Internal error getting user.")
	}
	log, ok := ctx.Value("request-log").(logging.LogAccumulator)
	if !ok {
		return di, huma.Error500InternalServerError(
			"Internal error while getting request log.")
	}
	di, err = a.checkAccess(log, user, deviceId)
	if err != nil {
		return
	}
	if !authstore.HasPermission(authstore.Role(user.Role), permission) {
		return di, huma.Error401Unauthorized("Access denied.")
	}
	return
}

func (a *Api) user(ctx context.Context) (authstore.UserInfo, error) {
	user, ok := ctx.Value("user").(authstore.UserInfo)
	if !ok {
		return user, huma.Error500InternalServerError(
			"Internal error getting user.")
	}
	return user, nil
}

func (a *Api) getSensorDataBoundary(ctx context.Context, in *data.DataBoundaryRequest) (
	db data.DataBoundary, err error) {
	di, err := a.deviceInfoIfAccessAndPermission(ctx, in.DeviceId, authstore.GetDataBoundary)
	if err != nil {
		return db, err
	}
	di.Timezone, err = in.Timezone.Location()
	if err != nil {
		return db, huma.Error400BadRequest(
			"Invalid location. Please try a different IANA Timezone.")
	}
	log, ok := ctx.Value("request-log").(logging.LogAccumulator)
	if !ok {
		return db, huma.Error500InternalServerError(
			"Internal error while getting request log.")
	}
	db, err = a.DataFetcher.GetDataBoundary(di)
	if err != nil {
		if errors.Is(err, sensor.NoData) {
			return db, err
		}
		log.AddMetadata(logging.Metadata{
			"source":        {"getSensorDataBoundary.dataFetcher.getDataBoundary"},
			"error.message": {err.Error()}})
		return db, huma.Error500InternalServerError(
			"Internal error getting DataBoundary.")
	}
	return
}

func (a *Api) getLocation(ctx context.Context, in *data.DeviceLocationRequest) (
	loc data.DeviceLocationResponse, err error) {
	di, err := a.deviceInfoIfAccessAndPermission(ctx, in.DeviceId, authstore.GetDataBoundary)
	if err != nil {
		return loc, err
	}
	log, ok := ctx.Value("request-log").(logging.LogAccumulator)
	if !ok {
		return loc, huma.Error500InternalServerError(
			"Internal error while getting request log.")
	}
	loc, err = a.DataFetcher.GetLocation(di)
	if err != nil {
		if errors.Is(err, sensor.NoLocation) {
			return loc, err
		}
		log.AddMetadata(logging.Metadata{
			"source":        {"getLocation.dataFetcher.getLocation"},
			"error.message": {err.Error()}})
		return loc, huma.Error500InternalServerError(
			"Internal error getting Location.")
	}
	return
}
