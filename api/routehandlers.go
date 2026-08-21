package api

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humamux"
	"github.com/datafarm-software/datafarm-api/api/authstore"
	"github.com/datafarm-software/datafarm-api/api/datafetcher"
	deviceinfo "github.com/datafarm-software/datafarm-api/api/device-info"
	"github.com/datafarm-software/datafarm-api/api/tokenprovider"
)

func (a *Api) GetSensorData(ctx context.Context,
	in *datafetcher.SensorDataRequest) (out *datafetcher.SensorDataResponse, err error) {
	sensorData, err := a.getSensorData(ctx, in)
	if err != nil {
		return nil, err
	}
	if len(sensorData) < 1 {
		return &datafetcher.SensorDataResponse{Status: http.StatusNoContent}, nil
	}
	return &datafetcher.SensorDataResponse{
		Status: http.StatusOK,
		Body:   sensorData,
	}, nil
}

func (a *Api) VerifyToken(humaCtx huma.Context, next func(huma.Context)) {
	_, w := humamux.Unwrap(humaCtx)
	authHeader := humaCtx.Header("Authorization")
	if authHeader == "" {
		http.Error(w, "No Authorization Header Provided.", http.StatusBadRequest)
		return
	}
	parts := strings.Split(authHeader, "Bearer")
	if len(parts) != 2 {
		http.Error(w, "Invalid Authorization Header Format.", http.StatusBadRequest)
		return
	}
	var lr tokenprovider.LoginResponse
	lr.Body = strings.TrimSpace(parts[1])
	lr.Body = strings.Trim(lr.Body, `"`)
	if !a.TokenProvider.ValidToken(lr) {
		if err := a.AuthStore.DeleteToken(authstore.UserToken{Token: lr.Body}); err != nil {
			logMetadata(humaCtx.Context(), map[string]string{"authstore.error.message": err.Error()})
			http.Error(w,
				`Your token is invalid. Please login again. 
				There was an internal error while deleting the invalid token.`,
				http.StatusInternalServerError)
			return
		}
		http.Error(w, "Your token is invalid. Please login again.", http.StatusUnauthorized)
		return
	}
	user, err := a.AuthStore.GetUser(lr.Body)
	if err != nil {
		logMetadata(humaCtx.Context(), map[string]string{"authstore.error.message": fmt.Sprintf("getting user: %v", err)})
		http.Error(w, "Internal error while getting user information.",
			http.StatusInternalServerError)
		return
	}
	logMetadata(humaCtx.Context(),
		map[string]string{"client.username": user.Username,
			"client.company": user.Company,
			"client.network": user.Network,
		})
	next(huma.WithValue(humaCtx, "user", user))
}

func (a *Api) Login(ctx context.Context,
	ar *tokenprovider.LoginRequest) (*tokenprovider.LoginResponse, error) {
	parts := strings.Split(ar.Auth, " ")
	if len(parts) != 2 || parts[0] != "Basic" {
		return nil, huma.Error400BadRequest(
			"Authorization header must follow the basic format: 'Basic base64(username:password)'")
	}
	authBytes, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		logMetadata(ctx, map[string]string{"domain.error.message": fmt.Sprintf("base64 decode: %v", err)})
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
		log.Printf("verifyCredentials error: %v", err)
		return nil, huma.Error401Unauthorized("Bad credentials provided.")
	}
	ut, err := a.AuthStore.GetToken(username)
	if err != nil {
		if !errors.Is(err, authstore.NotLoggedIn) {
			logMetadata(ctx, map[string]string{"authstore.error.message": fmt.Sprintf("getting token: %v", err)})
			return nil, huma.Error500InternalServerError(
				"Internal error checking if user is logged in.")
		}
	}
	if ut.Token != "" {
		return &tokenprovider.LoginResponse{Body: ut.Token}, nil
	}
	ut, err = a.TokenProvider.GenerateToken(username)
	if err != nil {
		logMetadata(ctx, map[string]string{"tokenprovider.error.message": fmt.Sprintf("generate token: %v", err)})
		return nil, huma.Error500InternalServerError(
			"Internal error generating an access token.")
	}
	if err = a.AuthStore.StoreToken(ut); err != nil {
		logMetadata(ctx, map[string]string{"authstore.error.message": fmt.Sprintf("store token: %v", err)})
		return nil, huma.Error500InternalServerError(
			"Internal error linking the token to the user.")
	}
	a.Meter.ActiveUsersCountAdd(1)
	return &tokenprovider.LoginResponse{Body: ut.Token}, nil
}

func (a *Api) GetQueryFields(ctx context.Context, in *deviceinfo.QueryFieldsRequest) (
	*deviceinfo.QueryFieldsResponse, error) {
	user, ok := ctx.Value("user").(authstore.UserInfo)
	if !ok {
		logMetadata(ctx, map[string]string{
			"domain.error.message": "authstore.UserInfo not found in context"})
		return nil, huma.Error500InternalServerError(
			"Internal error getting user.")
	}
	_, code, err := a.checkAccessToDevice(in.DeviceId, user)
	if err != nil {
		switch code {
		case http.StatusUnauthorized:
			return nil, huma.Error401Unauthorized(
				"Unauthorized access to this device.")
		case http.StatusNotFound:
			return nil, huma.Error404NotFound(
				"Device Not Found.")
		default:
			logMetadata(ctx, map[string]string{"deviceinfo.error.message": err.Error()})
			return nil, huma.Error500InternalServerError(
				"Internal error checking acess to DeviceId.")
		}
	}
	if !authstore.HasPermission(authstore.Role(user.Role),
		authstore.GetAllQueryFields) {
		return nil, huma.Error401Unauthorized("Access denied to QueryFields.")
	}
	queryFields, err := a.DeviceInfo.GetQueryFields(in.DeviceId)
	if err != nil {
		logMetadata(ctx, map[string]string{"deviceinfo.error.message": fmt.Sprintf(
			"get queryfields: %v", err)})
		return nil, huma.Error500InternalServerError(
			"Internal error while getting queryfields.")
	}
	return &deviceinfo.QueryFieldsResponse{Body: queryFields}, nil
}

func (a *Api) BatchGetSensorData(ctx context.Context,
	in *struct {
		Body datafetcher.BatchSensorDataRequest
	}) (*struct {
	Body *datafetcher.BatchSensorDataResponse
}, error) {
	logRequest(ctx, in.Body)
	var dataReq *datafetcher.SensorDataRequest
	var deviceErr datafetcher.SensorDataError
	var sds datafetcher.SensorDataSlice
	var err error
	errSlice := make([]datafetcher.SensorDataError, 0, len(in.Body.Hardware))
	resultSlice := make(datafetcher.SensorDataSlice, 0, len(in.Body.Hardware))
	for _, hw := range in.Body.Hardware {
		dataReq = &datafetcher.SensorDataRequest{
			Hardware:  hw,
			TimeFrame: in.Body.TimeFrame,
		}
		sds, err = a.getSensorData(ctx, dataReq)
		if err == nil {
			resultSlice = append(resultSlice, sds...)
		} else {
			deviceErr.DeviceId = hw.DeviceId
			deviceErr.Error = err.Error()
			errSlice = append(errSlice, deviceErr)
		}
	}
	return &struct {
		Body *datafetcher.BatchSensorDataResponse
	}{
		Body: &datafetcher.BatchSensorDataResponse{
			Results: resultSlice,
			Errors:  errSlice,
		},
	}, nil
}

func (a *Api) BatchGetQueryFields(ctx context.Context,
	in *deviceinfo.BatchQueryFieldsRequest) (*struct {
	Body deviceinfo.BatchQueryFieldsResponse
}, error) {
	var qr deviceinfo.QueryFieldsRequest
	var dataResp *deviceinfo.QueryFieldsResponse
	var deviceErr deviceinfo.QueryFieldsError
	var err error
	errSlice := make([]deviceinfo.QueryFieldsError, 0, len(in.Body.DeviceIds))
	resultSlice := make([]deviceinfo.QueryFields, 0, len(in.Body.DeviceIds))
	for _, deviceId := range in.Body.DeviceIds {
		qr = deviceinfo.QueryFieldsRequest{
			DeviceId: deviceId,
		}
		dataResp, err = a.GetQueryFields(ctx, &qr)
		if err == nil {
			resultSlice = append(resultSlice, dataResp.Body)
		} else {
			deviceErr.DeviceId = deviceId
			deviceErr.Error = err.Error()
			errSlice = append(errSlice, deviceErr)
		}
	}
	return &struct {
		Body deviceinfo.BatchQueryFieldsResponse
	}{
		Body: deviceinfo.BatchQueryFieldsResponse{
			Results: resultSlice,
			Errors:  errSlice,
		},
	}, nil
}

func (a *Api) GetDeviceIds(ctx context.Context, _ *struct{}) (
	*deviceinfo.DeviceIdsResponse, error) {
	user, ok := ctx.Value("user").(authstore.UserInfo)
	if !ok {
		logMetadata(ctx, map[string]string{
			"domain.error.message": "authstore.UserInfo not found in context"})
		return nil, huma.Error500InternalServerError(
			"Internal error getting user.")
	}
	sr := deviceinfo.ScopeRestriction{
		Company: user.Company,
		Network: user.Network,
	}
	switch authstore.Role(user.Role) {
	case authstore.User:
		sr.Scope = deviceinfo.DevicesInCompanyInNetwork
	case authstore.NetworkUser:
		sr.Scope = deviceinfo.DevicesInNetwork
	case authstore.Admin:
		sr.Scope = deviceinfo.AllDevices
	default:
		logMetadata(ctx, map[string]string{"domain.error.message": fmt.Sprintf("unknown user role: %v", user.Role)})
		return nil, huma.Error500InternalServerError(
			"Internal error determining user role.")
	}
	userDevices, err := a.DeviceInfo.GetDevices(sr)
	if err != nil {
		logMetadata(ctx, map[string]string{"deviceinfo.error.message": fmt.Sprintf("get devices: %v", err)})
		return nil, huma.Error500InternalServerError(
			"Internal error getting DeviceIds.")
	}
	return &deviceinfo.DeviceIdsResponse{
		Body: userDevices,
	}, nil
}

func (a *Api) GetSensorDataBoundary(ctx context.Context, in *datafetcher.DataBoundaryRequest) (
	*datafetcher.DataBoundaryResponse, error) {
	user, ok := ctx.Value("user").(authstore.UserInfo)
	if !ok {
		logMetadata(ctx, map[string]string{
			"domain.error.message": "authstore.UserInfo not found in context"})
		return nil, huma.Error500InternalServerError(
			"Internal error getting user.")
	}
	di, code, err := a.checkAccessToDevice(in.DeviceId, user)
	if err != nil {
		switch code {
		case http.StatusUnauthorized:
			return nil, huma.Error401Unauthorized(
				"Unauthorized access to this device.")
		case http.StatusNotFound:
			return nil, huma.Error404NotFound(
				"Device Not Found.")
		default:
			logMetadata(ctx, map[string]string{"deviceinfo.error.message": err.Error()})
			return nil, huma.Error500InternalServerError(
				"Internal error checking acess to DeviceId.")
		}
	}
	if !authstore.HasPermission(authstore.Role(user.Role),
		authstore.GetDataBoundary) {
		return nil, huma.Error401Unauthorized("Access denied to DataBoundary.")
	}
	di.Timezone, err = in.Timezone.Location()
	if err != nil {
		return nil, huma.Error400BadRequest(
			"Invalid location. Please try a different IANA Timezone.")
	}
	dataBoundary, err := a.DataFetcher.GetDataBoundary(di)
	if err != nil {
		logMetadata(ctx, map[string]string{"datafetcher.error.message": fmt.Sprintf("getting data boundary: %v", err)})
		return nil, huma.Error500InternalServerError(
			"Internal error getting DataBoundary.")
	}
	return &datafetcher.DataBoundaryResponse{Body: dataBoundary}, nil
}

func (a *Api) BatchGetSensorDataBoundary(ctx context.Context,
	in *struct {
		Body datafetcher.BatchDataBoundaryRequest
	}) (
	*struct {
		Body datafetcher.BatchDataBoundaryResponse
	}, error) {
	var qr datafetcher.DataBoundaryRequest
	var dataResp *datafetcher.DataBoundaryResponse
	var deviceErr datafetcher.DataBoundaryError
	var err error
	errSlice := make([]datafetcher.DataBoundaryError, 0, len(in.Body.DeviceIds))
	resultSlice := make([]datafetcher.DataBoundary, 0, len(in.Body.DeviceIds))
	for _, deviceId := range in.Body.DeviceIds {
		qr = datafetcher.DataBoundaryRequest{
			DeviceId: deviceId,
			Timezone: in.Body.Timezone,
		}
		dataResp, err = a.GetSensorDataBoundary(ctx, &qr)
		if err == nil {
			resultSlice = append(resultSlice, dataResp.Body)
		} else {
			deviceErr.DeviceId = deviceId
			deviceErr.Error = err.Error()
			errSlice = append(errSlice, deviceErr)
		}
	}
	return &struct {
		Body datafetcher.BatchDataBoundaryResponse
	}{
		Body: datafetcher.BatchDataBoundaryResponse{
			Results: resultSlice,
			Errors:  errSlice,
		},
	}, nil
}
