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
	sensorData, err := a.getSensorData(ctx, *in)
	if err != nil {
		if errors.Is(err, sensor.NotFound) {
			return nil, huma.Error404NotFound("Not Found.")
		}
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
	sensorData, err := a.getLatestSensorData(ctx, *in)
	if err != nil {
		if errors.Is(err, sensor.NotFound) {
			return nil, huma.Error404NotFound("Not Found.")
		}
		if errors.Is(err, sensor.NoData) {
			return &data.LatestSensorDataResponse{
				Status: http.StatusNoContent,
			}, nil
		}
		return nil, err
	}
	if len(sensorData) != 1 {
		return nil, huma.Error500InternalServerError(
			"Unexpected error while getting Latest SensorData.")
	}
	return &data.LatestSensorDataResponse{
		Status: http.StatusOK,
		Body:   sensorData[0],
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
	batch, err := sensor.BatchFactory(ctx, in.Body.Hardware,
		func(hw sensor.Hardware) data.SensorDataRequest {
			return data.SensorDataRequest{
				Hardware: hw, TimeFrame: data.TimeFrame{
					Start:    in.Body.Start,
					Stop:     in.Body.Stop,
					Timezone: in.Body.Timezone,
				},
			}
		},
		a.getSensorData,
	)
	if err != nil {
		if errors.Is(err, sensor.NotFound) {
			return nil, huma.Error404NotFound("No DeviceIds Found.")
		}
		if errors.Is(err, sensor.NoConnection) {
			return nil, huma.Error500InternalServerError("Database Disconnected.")
		}
		return nil, huma.Error500InternalServerError(
			"Unexpected internal error while getting SensorData.")
	}
	resp := &struct {
		Status int
		Body   *data.BatchSensorDataResponse
	}{
		Status: http.StatusOK,
		Body: &data.BatchSensorDataResponse{
			Results: batch.Results,
			Errors:  batch.Errors,
		},
	}
	if len(batch.Results) < 1 && batch.OnlyDataMissingErrors {
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
	batch, err := sensor.BatchFactory(ctx, in.Body.Hardware,
		func(hw sensor.Hardware) data.LatestSensorDataRequest {
			return data.LatestSensorDataRequest{
				Hardware: hw,
				Timezone: in.Body.Timezone,
			}
		},
		a.getLatestSensorData,
	)
	if err != nil {
		if errors.Is(err, sensor.NotFound) {
			return nil, huma.Error404NotFound("No DeviceIds Found.")
		}
		if errors.Is(err, sensor.NoConnection) {
			return nil, huma.Error500InternalServerError("Database Disconnected.")
		}
		return nil, huma.Error500InternalServerError(
			"Unexpected internal error with batch request.")
	}
	resp := &struct {
		Status int
		Body   *data.BatchSensorDataResponse
	}{
		Status: http.StatusOK,
		Body: &data.BatchSensorDataResponse{
			Results: batch.Results,
			Errors:  batch.Errors,
		},
	}
	if len(batch.Results) < 1 && batch.OnlyDataMissingErrors {
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

func (a *Api) GetQueryFields(ctx context.Context, in *sensor.DeviceIdParam) (
	*info.QueryFieldsResponse, error) {
	logFromTag(ctx, in)
	queryFields, err := a.getQueryFields(ctx, *in)
	if err != nil {
		if errors.Is(err, sensor.NoConnection) {
			return nil, huma.Error500InternalServerError(
				"Database Disconnected.")
		}
		if errors.Is(err, sensor.NotFound) {
			return nil, huma.Error404NotFound("Not Found.")
		}
		return nil, err
	}
	if len(queryFields) != 1 {
		return nil, huma.Error500InternalServerError(
			"Internal error while getting QueryFields")
	}
	return &info.QueryFieldsResponse{Body: queryFields[0]}, nil
}

func (a *Api) BatchGetQueryFields(ctx context.Context,
	in *info.BatchQueryFieldsRequest) (*struct {
	Body info.BatchQueryFieldsResponse
}, error) {
	logFromTag(ctx, in)
	batch, err := sensor.BatchFactory(ctx, in.Body.DeviceIds,
		func(d sensor.DeviceId) sensor.DeviceIdParam {
			return sensor.DeviceIdParam{DeviceId: d}
		},
		a.getQueryFields,
	)
	if err != nil {
		if errors.Is(err, sensor.NoConnection) {
			return nil, huma.Error500InternalServerError("Database Disconnected.")
		}
		if errors.Is(err, sensor.NotFound) {
			return nil, huma.Error404NotFound("DeviceIds Not Found.")
		}
		return nil, huma.Error500InternalServerError(
			"Unexpected internal error while getting SensorData.")
	}
	return &struct {
		Body info.BatchQueryFieldsResponse
	}{
		Body: info.BatchQueryFieldsResponse{
			Results: batch.Results,
			Errors:  batch.Errors,
		},
	}, nil
	// if len(batch.Results) < 1 && batch.OnlyDataMissingErrors {
	// 	resp.Status = http.StatusNoContent
	// 	resp.Body = nil
	// }
	// return resp, nil
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
		if errors.Is(err, sensor.NoConnection) {
			return nil, huma.Error500InternalServerError(
				"Database disconnected.")
		}
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
	db, err := a.getDataBoundary(ctx, *in)
	if err != nil {
		if errors.Is(err, sensor.NoConnection) {
			return nil, huma.Error500InternalServerError(
				"Database Disconnected.")
		}
		if errors.Is(err, sensor.NoData) {
			return &data.DataBoundaryResponse{Status: http.StatusNoContent}, nil
		}
		if errors.Is(err, sensor.NotFound) {
			return nil, huma.Error404NotFound("Not Found.")
		}
		return nil, err
	}
	if len(db) != 1 {
		return nil, huma.Error500InternalServerError(
			"Unexpected internal error while getting DataBoundary.")
	}
	return &data.DataBoundaryResponse{Status: http.StatusOK, Body: db[0]}, nil
}

func (a *Api) BatchGetDataBoundary(ctx context.Context,
	in *struct {
		Body data.BatchDataBoundaryRequest
	}) (
	*struct {
		Status int
		Body   data.BatchDataBoundaryResponse
	}, error) {
	logFromTag(ctx, in.Body)
	batch, err := sensor.BatchFactory(ctx, in.Body.DeviceIds,
		func(d sensor.DeviceId) data.DataBoundaryRequest {
			return data.DataBoundaryRequest{
				DeviceIdParam: sensor.DeviceIdParam{DeviceId: d},
				Timezone:      in.Body.Timezone,
			}
		},
		a.getDataBoundary)
	if err != nil {
		if errors.Is(err, sensor.NoConnection) {
			return nil, huma.Error500InternalServerError("Database Disconnected.")
		}
		if errors.Is(err, sensor.NotFound) {
			return nil, huma.Error404NotFound("Not Found.")
		}
		if !errors.Is(err, sensor.NoData) {
			return nil, huma.Error500InternalServerError(
				"Unexpected internal error while getting DataBoundary.")
		}
	}
	resp := &struct {
		Status int
		Body   data.BatchDataBoundaryResponse
	}{
		Status: http.StatusOK,
		Body: data.BatchDataBoundaryResponse{
			Results: batch.Results,
			Errors:  batch.Errors,
		},
	}
	if len(batch.Results) < 1 && batch.OnlyDataMissingErrors {
		resp.Status = http.StatusNoContent
		resp.Body = data.BatchDataBoundaryResponse{}
	}
	return resp, nil
}

func (a *Api) GetLocation(ctx context.Context, in *sensor.DeviceIdParam) (
	*struct {
		Status int
		Body   data.DeviceLocationResponse
	}, error) {
	logFromTag(ctx, in)
	loc, err := a.getLocation(ctx, *in)
	if err != nil {
		if errors.Is(err, sensor.NoConnection) {
			return nil, huma.Error500InternalServerError(
				"Database Disconnected.")
		}
		if errors.Is(err, sensor.NotFound) {
			return nil, huma.Error404NotFound("Not Found.")
		}
		if errors.Is(err, sensor.NoLocation) {
			return &struct {
				Status int
				Body   data.DeviceLocationResponse
			}{http.StatusNoContent, data.DeviceLocationResponse{}}, nil
		}
		return nil, err
	}
	if len(loc) != 1 {
		return nil, huma.Error500InternalServerError(
			"Unexpected internal error while getting Location.")
	}
	return &struct {
		Status int
		Body   data.DeviceLocationResponse
	}{http.StatusOK, loc[0]}, nil
}

func (a *Api) BatchGetLocation(ctx context.Context, in *struct {
	Body data.BatchLocationRequest
}) (*struct {
	Status int
	Body   data.BatchLocationResponse
}, error) {
	logFromTag(ctx, in.Body)
	batch, err := sensor.BatchFactory(ctx, in.Body.DeviceIds,
		func(d sensor.DeviceId) sensor.DeviceIdParam {
			return sensor.DeviceIdParam{DeviceId: d}
		},
		a.getLocation)
	if err != nil {
		if errors.Is(err, sensor.NotFound) {
			return nil, huma.Error404NotFound("DeviceIds Not Found.")
		}
		if errors.Is(err, sensor.NoConnection) {
			return nil, huma.Error500InternalServerError("Database Disconnected.")
		}
		if !errors.Is(err, sensor.NoLocation) {
			return nil, huma.Error500InternalServerError(
				"Unexpected internal error while getting Locations.")
		}
	}
	resp := &struct {
		Status int
		Body   data.BatchLocationResponse
	}{
		Status: http.StatusOK,
		Body: data.BatchLocationResponse{
			Results: batch.Results,
			Errors:  batch.Errors,
		},
	}
	if len(batch.Results) < 1 && batch.OnlyDataMissingErrors {
		resp.Status = http.StatusNoContent
		resp.Body = data.BatchLocationResponse{}
	}
	return resp, nil
}
