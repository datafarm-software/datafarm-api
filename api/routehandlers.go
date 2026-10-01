package api

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humamux"
	"github.com/datafarm-software/datafarm-api/api/authstore"
	"github.com/datafarm-software/datafarm-api/api/sensor"
	"github.com/datafarm-software/datafarm-api/api/sensor/data"
	"github.com/datafarm-software/datafarm-api/api/sensor/info"
	"github.com/datafarm-software/datafarm-api/api/tokenprovider"
	"github.com/datafarm-software/telemetry/logging"
)

func (a *Api) GetSensorData(ctx context.Context,
	in *data.SensorDataRequest) (out *data.SensorDataResponse, err error) {
	logFromTag(ctx, in)
	sensorData, err := a.getSensorData(ctx, in)
	if err != nil {
		if errors.Is(err, sensor.NoData) {
			return &data.SensorDataResponse{Status: http.StatusNoContent}, nil
		}
		return nil, err
	}
	if len(sensorData) < 1 {
		return &data.SensorDataResponse{Status: http.StatusNoContent}, nil
	}
	return &data.SensorDataResponse{
		Status: http.StatusOK,
		Body:   sensorData,
	}, nil
}

func (a *Api) GetLatestSensorData(ctx context.Context,
	in *data.LatestSensorDataRequest) (
	out *data.LatestSensorDataResponse, err error) {
	logFromTag(ctx, in)
	sensorData, err := a.getLatestSensorData(ctx, in)
	if err != nil {
		if errors.Is(err, sensor.NoData) {
			return &data.LatestSensorDataResponse{
				Status: http.StatusNoContent,
			}, nil
		}
		return nil, err
	}
	return &data.LatestSensorDataResponse{
		Status: http.StatusOK,
		Body:   sensorData,
	}, nil
}

func (a *Api) BatchGetSensorData(ctx context.Context,
	in *struct {
		Body data.BatchSensorDataRequest
	}) (*struct {
	Status int
	Body   *data.BatchSensorDataResponse
}, error) {
	logFromTag(ctx, in.Body)
	var dataReq *data.SensorDataRequest
	var deviceErr data.SensorDataError
	var sds data.SensorDataSlice
	var err error
	var onlyDataMissingErrors = true
	errSlice := make([]data.SensorDataError, 0, len(in.Body.Hardware))
	resultSlice := make(data.SensorDataSlice, 0, len(in.Body.Hardware))
	for _, hw := range in.Body.Hardware {
		dataReq = &data.SensorDataRequest{
			Hardware:  hw,
			TimeFrame: in.Body.TimeFrame,
		}
		sds, err = a.getSensorData(ctx, dataReq)
		if err == nil {
			resultSlice = append(resultSlice, sds...)
		} else {
			if !errors.Is(err, sensor.NoData) {
				onlyDataMissingErrors = false
				if errors.Is(err, sensor.NoConnection) {
					break
				}
			}
			deviceErr.DeviceId = hw.DeviceIdParam.DeviceId
			deviceErr.Error = err.Error()
			errSlice = append(errSlice, deviceErr)
		}
		err = nil
	}
	resp := &struct {
		Status int
		Body   *data.BatchSensorDataResponse
	}{
		Status: http.StatusOK,
		Body: &data.BatchSensorDataResponse{
			Results: resultSlice,
			Errors:  errSlice,
		},
	}
	if errors.Is(err, sensor.NoConnection) {
		return nil, huma.Error500InternalServerError("Database Disconnected.")
	}
	if len(resultSlice) < 1 && onlyDataMissingErrors {
		resp.Status = http.StatusNoContent
		resp.Body = nil
	}
	return resp, nil
}

func (a *Api) BatchGetLatestSensorData(ctx context.Context,
	in *struct {
		Body data.BatchLatestSensorDataRequest
	}) (*struct {
	Status int
	Body   *data.BatchSensorDataResponse
}, error) {
	logFromTag(ctx, in.Body)
	var dataReq *data.LatestSensorDataRequest
	var deviceErr data.SensorDataError
	var sds data.SensorDataSlice
	var err error
	onlyDataMissingErrors := true
	errSlice := make([]data.SensorDataError, 0, len(in.Body.Hardware))
	resultSlice := make(data.SensorDataSlice, 0, len(in.Body.Hardware))
	for _, hw := range in.Body.Hardware {
		dataReq = &data.LatestSensorDataRequest{
			Hardware: hw,
			Timezone: in.Body.Timezone,
		}
		sds, err = a.getLatestSensorData(ctx, dataReq)
		if err == nil {
			resultSlice = append(resultSlice, sds...)
		} else {
			if errors.Is(err, sensor.NoConnection) {
				break
			}
			if !errors.Is(err, sensor.NoData) {
				onlyDataMissingErrors = false
			}
			deviceErr.DeviceId = hw.DeviceIdParam.DeviceId
			deviceErr.Error = err.Error()
			errSlice = append(errSlice, deviceErr)
		}
	}
	if errors.Is(err, sensor.NoConnection) {
		return nil, huma.Error500InternalServerError("Database Disconnected.")
	}
	resp := &struct {
		Status int
		Body   *data.BatchSensorDataResponse
	}{
		Status: http.StatusOK,
		Body: &data.BatchSensorDataResponse{
			Results: resultSlice,
			Errors:  errSlice,
		},
	}
	if len(resultSlice) < 1 && onlyDataMissingErrors {
		resp.Status = http.StatusNoContent
		resp.Body = nil
	}
	return resp, nil
}

func (a *Api) VerifyToken(humaCtx huma.Context, next func(huma.Context)) {
	_, w := humamux.Unwrap(humaCtx)
	authHeader := humaCtx.Header("Authorization")
	if authHeader == "" {
		a.httpErr(humaCtx, w, "No Authorization Header Provided.", http.StatusBadRequest)
		return
	}
	parts := strings.Split(authHeader, "Bearer")
	if len(parts) != 2 {
		a.httpErr(humaCtx, w, "Invalid Authorization Header Format.", http.StatusBadRequest)
		return
	}
	var lr tokenprovider.LoginResponse
	lr.Body = strings.TrimSpace(parts[1])
	lr.Body = strings.Trim(lr.Body, `"`)
	log, ok := humaCtx.Context().Value("request-log").(logging.LogAccumulator)
	if !ok {
		a.httpErr(humaCtx, w, "Internal error while getting request log.",
			http.StatusInternalServerError)
	}
	if !a.TokenProvider.ValidToken(lr) {
		if err := a.AuthStore.DeleteToken(authstore.UserToken{Token: lr.Body}); err != nil {
			log.AddMetadata(logging.Metadata{
				"source":        {"verifyToken.authStore.DeleteToken"},
				"error.message": {err.Error()}})
			a.httpErr(humaCtx, w,
				`Your token is invalid. Please login again. There was an internal error while deleting the invalid token.`,
				http.StatusInternalServerError)
			return
		}
		a.httpErr(humaCtx, w, "Your token is invalid. Please login again.", http.StatusUnauthorized)
		return
	}
	user, err := a.AuthStore.GetUser(lr.Body)
	if err != nil {
		log.AddMetadata(logging.Metadata{
			"source":        {"verifyToken.authStore.getUser"},
			"error.message": {fmt.Sprintf("getting user: %v", err)}})
		a.httpErr(humaCtx, w, "Internal error while getting user information.",
			http.StatusInternalServerError)
		return
	}
	log.AddMetadata(logging.Metadata{
		"client.username": {user.Username},
		"client.company":  {user.Company},
		"client.network":  {user.Network},
	})
	next(huma.WithValue(humaCtx, "user", user))
}

func (a *Api) Login(ctx context.Context,
	ar *tokenprovider.LoginRequest) (*tokenprovider.LoginResponse, error) {
	parts := strings.Split(ar.Auth, " ")
	logFromTag(ctx, ar)
	log, ok := ctx.Value("request-log").(logging.LogAccumulator)
	if !ok {
		huma.Error500InternalServerError("Internal error while getting request log.")
	}
	authBytes, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		log.AddMetadata(logging.Metadata{
			"source":        {"login.domain"},
			"error.message": {fmt.Sprintf("base64 decode: %v", err)}})
		return nil, huma.Error500InternalServerError(
			"Internal error decoding given base64.")
	}
	authInfo := strings.Split(string(authBytes), ":")
	if len(authInfo) != 2 {
		return nil, huma.Error400BadRequest("Invalid Basic format provided.")
	}
	username := authInfo[0]
	if ok := USERNAME_REGEX.MatchString(username); !ok {
		return nil, huma.Error400BadRequest("Username failed the regex.")
	}
	password := authInfo[1]
	if ok := UPPERCASE_REGEX.MatchString(password); !ok {
		return nil, huma.Error400BadRequest(
			"Password failed the regex.")
	}
	if ok := LOWERCASE_REGEX.MatchString(password); !ok {
		return nil, huma.Error400BadRequest(
			"Password failed the regex.")
	}
	if ok := NUMBER_REGEX.MatchString(password); !ok {
		return nil, huma.Error400BadRequest(
			"Password failed the regex.")
	}
	if ok := SPECIAL_CHARS_REGEX.MatchString(password); !ok {
		return nil, huma.Error400BadRequest(
			"Password failed the regex.")
	}
	if err = a.AuthStore.VerifyCredentials(username, password); err != nil {
		log.AddMetadata(logging.Metadata{
			"source":        {"login.authStore.verifyCredentials"},
			"error.message": {err.Error()}})
		return nil, huma.Error401Unauthorized("Bad credentials.")
	}
	ut, err := a.AuthStore.GetToken(username)
	if err != nil {
		if !errors.Is(err, authstore.NotLoggedIn) {
			log.AddMetadata(logging.Metadata{
				"source":        {"login.authStore.getToken"},
				"error.message": {err.Error()}})
			return nil, huma.Error500InternalServerError(
				"Internal error checking if user is logged in.")
		}
	}
	if ut.Token != "" {
		return &tokenprovider.LoginResponse{Body: ut.Token}, nil
	}
	ut, err = a.TokenProvider.GenerateToken(username)
	if err != nil {
		log.AddMetadata(logging.Metadata{
			"source":        {"login.tokenProvider.generateToken"},
			"error.message": {err.Error()}})
		return nil, huma.Error500InternalServerError(
			"Internal error generating an access token.")
	}
	if err = a.AuthStore.StoreToken(ut); err != nil {
		log.AddMetadata(logging.Metadata{
			"source":        {"login.authStore.storeToken"},
			"error.message": {err.Error()}})
		return nil, huma.Error500InternalServerError(
			"Internal error storing the token.")
	}
	a.Meter.ActiveUsersCountAdd(1)
	return &tokenprovider.LoginResponse{Body: ut.Token}, nil
}

func (a *Api) GetQueryFields(ctx context.Context, in *info.QueryFieldsRequest) (
	*info.QueryFieldsResponse, error) {
	logFromTag(ctx, in)
	queryFields, err := a.getQueryFields(ctx, in)
	if err != nil {
		return nil, err
	}
	return &info.QueryFieldsResponse{Body: queryFields}, nil
}

func (a *Api) BatchGetQueryFields(ctx context.Context,
	in *info.BatchQueryFieldsRequest) (*struct {
	Body info.BatchQueryFieldsResponse
}, error) {
	logFromTag(ctx, in)
	var qr info.QueryFieldsRequest
	var dataResp info.QueryFields
	var deviceErr info.QueryFieldsError
	var err error
	errSlice := make([]info.QueryFieldsError, 0, len(in.Body.DeviceIds))
	resultSlice := make([]info.QueryFields, 0, len(in.Body.DeviceIds))
	for _, deviceId := range in.Body.DeviceIds {
		qr = info.QueryFieldsRequest{
			DeviceId: deviceId,
		}
		dataResp, err = a.getQueryFields(ctx, &qr)
		if err == nil {
			resultSlice = append(resultSlice, dataResp)
		} else {
			deviceErr.DeviceId = deviceId
			deviceErr.Error = err.Error()
			errSlice = append(errSlice, deviceErr)
		}
	}
	return &struct {
		Body info.BatchQueryFieldsResponse
	}{
		Body: info.BatchQueryFieldsResponse{
			Results: resultSlice,
			Errors:  errSlice,
		},
	}, nil
}

func (a *Api) GetDeviceIds(ctx context.Context, _ *struct{}) (
	*struct {
		Status int
		Body   info.DeviceIdsResponse
	}, error) {
	user, ok := ctx.Value("user").(authstore.UserInfo)
	if !ok {
		return nil, huma.Error500InternalServerError(
			"Internal error getting user.")
	}
	sr := info.ScopeRestriction{
		Company: user.Company,
		Network: user.Network,
	}
	log, ok := ctx.Value("request-log").(logging.LogAccumulator)
	if !ok {
		huma.Error500InternalServerError("Internal error while getting request log.")
	}
	switch authstore.Role(user.Role) {
	case authstore.User:
		sr.Scope = info.DevicesInCompanyInNetwork
	case authstore.NetworkUser:
		sr.Scope = info.DevicesInNetwork
	case authstore.Admin:
		sr.Scope = info.AllDevices
	default:
		log.AddMetadata(logging.Metadata{
			"source":        {"getDeviceIds.domain"},
			"error.message": {fmt.Sprintf("unexpected user role: %v", user.Role)}})
		return nil, huma.Error500InternalServerError(
			"Internal error determining user role.")
	}
	userDevices, err := a.DeviceInfo.GetDevices(sr)
	if err != nil {
		log.AddMetadata(logging.Metadata{
			"source":        {"getDeviceIds.deviceInfo.getDevices"},
			"error.message": {err.Error()}})
		return nil, huma.Error500InternalServerError(
			"Internal error getting DeviceIds.")
	}
	resp := &struct {
		Status int
		Body   info.DeviceIdsResponse
	}{
		Status: http.StatusOK,
		Body:   info.DeviceIdsResponse{DeviceIds: userDevices},
	}
	if len(userDevices) < 1 {
		resp.Status = http.StatusNoContent
	}
	return resp, nil
}

func (a *Api) GetDataBoundary(ctx context.Context, in *data.DataBoundaryRequest) (
	*data.DataBoundaryResponse, error) {
	logFromTag(ctx, in)
	db, err := a.getSensorDataBoundary(ctx, in)
	if err != nil {
		if errors.Is(err, sensor.NoData) {
			return &data.DataBoundaryResponse{Status: http.StatusNoContent}, nil
		}
		return nil, err
	}
	return &data.DataBoundaryResponse{Status: http.StatusOK, Body: db}, nil
}

func (a *Api) BatchGetDataBoundary(ctx context.Context,
	in *struct {
		Body data.BatchDataBoundaryRequest
	}) (
	*struct {
		Body data.BatchDataBoundaryResponse
	}, error) {
	logFromTag(ctx, in.Body)
	var qr data.DataBoundaryRequest
	var dataResp data.DataBoundary
	var deviceErr data.BatchError
	var err error
	errSlice := make([]data.BatchError, 0, len(in.Body.DeviceIds))
	resultSlice := make([]data.DataBoundary, 0, len(in.Body.DeviceIds))
	for _, deviceId := range in.Body.DeviceIds {
		qr = data.DataBoundaryRequest{
			data.DeviceIdParam{DeviceId: deviceId}, in.Body.Timezone,
		}
		dataResp, err = a.getSensorDataBoundary(ctx, &qr)
		if err == nil {
			resultSlice = append(resultSlice, dataResp)
		} else {
			deviceErr.DeviceId = deviceId
			deviceErr.Error = err.Error()
			errSlice = append(errSlice, deviceErr)
		}
	}
	return &struct {
		Body data.BatchDataBoundaryResponse
	}{
		Body: data.BatchDataBoundaryResponse{
			Results: resultSlice,
			Errors:  errSlice,
		},
	}, nil
}

func (a *Api) GetLocation(ctx context.Context, in *data.DeviceLocationRequest) (
	*struct {
		Status int
		Body   data.DeviceLocationResponse
	}, error) {
	logFromTag(ctx, in)
	loc, err := a.getLocation(ctx, in)
	if err != nil {
		if !errors.Is(err, sensor.NoLocation) {
			return nil, err
		}
		return &struct {
			Status int
			Body   data.DeviceLocationResponse
		}{http.StatusNoContent, data.DeviceLocationResponse{}}, nil
	}
	return &struct {
		Status int
		Body   data.DeviceLocationResponse
	}{http.StatusOK, loc}, nil
}

func (a *Api) BatchGetLocation(ctx context.Context, in *struct {
	Body data.BatchLocationRequest
}) (*struct {
	Body data.BatchLocationResponse
}, error) {
	logFromTag(ctx, in.Body)
	var lr data.DeviceLocationRequest
	var dataResp data.DeviceLocationResponse
	var deviceErr data.BatchError
	var err error
	errSlice := make([]data.BatchError, 0, len(in.Body.DeviceIds))
	resultSlice := make([]data.DeviceLocationResponse, 0, len(in.Body.DeviceIds))
	for _, deviceId := range in.Body.DeviceIds {
		lr.DeviceId = deviceId
		dataResp, err = a.getLocation(ctx, &lr)
		if err == nil {
			resultSlice = append(resultSlice, dataResp)
		} else {
			deviceErr.DeviceId = deviceId
			deviceErr.Error = err.Error()
			errSlice = append(errSlice, deviceErr)
		}
	}
	return &struct {
		Body data.BatchLocationResponse
	}{
		Body: data.BatchLocationResponse{
			Results: resultSlice,
			Errors:  errSlice,
		},
	}, nil
}
