package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	contextutils "github.com/fabric8-services/fabric8-wit/api/context_utils"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/gin-gonic/gin"
	"github.com/google/jsonapi"
	uuid "github.com/satori/go.uuid"
)

func baseURL(req *http.Request) string {
	url := req.URL
	return fmt.Sprintf("%[1]s://%[2]s", url.Scheme, url.Host)
}

// OK Responds with a '200 OK' response
func OK(ctx *gin.Context, result interface{}) error {
	ctx.Status(http.StatusOK)
	ctx.Header("Content-Type", jsonapi.MediaType)
	switch result := result.(type) {
	case jsonapi.ManyPayload:
		ctx.Header("Content-Type", jsonapi.MediaType)
		if err := json.NewEncoder(ctx.Writer).Encode(result); err != nil {
			contextutils.AbortWithError(ctx, err)
			return err
		}
	default:
		if err := jsonapi.MarshalPayload(ctx.Writer, result); err != nil {
			contextutils.AbortWithError(ctx, err)
			return err
		}
	}
	return nil
}

// NotModified Responds with a '304 NotModified' response
func NotModified(ctx *gin.Context) error {
	ctx.Status(http.StatusNotModified)
	return nil
}

// Created Responds with a '201 Created' response with response body and a 'Location' header
func Created(ctx *gin.Context, result interface{}, location string) {
	ctx.Status(http.StatusCreated)
	ctx.Header("Content-Type", jsonapi.MediaType)
	ctx.Header("Location", location)
	if err := jsonapi.MarshalPayload(ctx.Writer, result); err != nil {
		contextutils.AbortWithError(ctx, err)
		return
	}
}

// GetParamAsString returns the query parameter as a string from its given key
// return nil if the request parameter was not present
func GetParamAsString(ctx *gin.Context, key string) *string {
	value := ctx.Param(key)
	if value == "" {
		return nil
	}
	return &value
}

// GetParamAsInt returns the query parameter as a integer from its given key
// return nil if the query parameter was not present
// calls `ctx.AbortWithError` if an error occurred while converting the request parameter value into an integer
func GetParamAsInt(ctx *gin.Context, key string) (*int, error) {
	value := ctx.Param(key)
	if value == "" {
		return nil, nil
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		ctx.AbortWithError(http.StatusBadRequest, errors.NewBadParameterError(fmt.Sprintf("request parameter '%s' is not a valid integer", key), value))
		return nil, err
	}
	return &intValue, err
}

// GetParamAsUUID returns the query parameter as a int from its given key
// return nil if the query parameter was not present
// calls `ctx.AbortWithError` if an error occurred while converting the query parameter value into an UUID
func GetParamAsUUID(ctx *gin.Context, key string) (*uuid.UUID, error) {
	value := ctx.Param(key)
	if value == "" {
		return nil, nil
	}
	uuidValue, err := uuid.FromString(ctx.Param(key))
	if err != nil {
		ctx.AbortWithError(http.StatusBadRequest, errors.NewBadParameterError(fmt.Sprintf("request parameter '%s' is not a valid UUID", key), value))
		return nil, err
	}
	return &uuidValue, err
}

// GetQueryParamAsString returns the query parameter as a string from its given key
// return nil if the query parameter was not present
func GetQueryParamAsString(ctx *gin.Context, key string) *string {
	value := ctx.Query(key)
	if value == "" {
		return nil
	}
	return &value
}

// GetQueryParamAsInt returns the query parameter as a integer from its given key
// return nil if the query parameter was not present
// calls `ctx.AbortWithError` if an error occurred while converting the query parameter value into an integer
func GetQueryParamAsInt(ctx *gin.Context, key string) (*int, error) {
	value := ctx.Query(key)
	if value == "" {
		return nil, nil
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		ctx.AbortWithError(http.StatusBadRequest, errors.NewBadParameterError(fmt.Sprintf("request parameter '%s' is not a valid integer", key), value))
		return nil, err
	}
	return &intValue, err
}

// GetQueryParamAsUUID returns the query parameter as a int from its given key
// return nil if the query parameter was not present
// calls `ctx.AbortWithError` if an error occurred while converting the query parameter value into an UUID
func GetQueryParamAsUUID(ctx *gin.Context, key string) (*uuid.UUID, error) {
	value := ctx.Query(key)
	if value == "" {
		return nil, nil
	}
	uuidValue, err := uuid.FromString(ctx.Param(key))
	if err != nil {
		ctx.AbortWithError(http.StatusBadRequest, errors.NewBadParameterError(fmt.Sprintf("query parameter '%s' is not a valid UUID", key), value))
	}
	return &uuidValue, err
}

// GetQueryParamAsBool returns the query parameter as a boolean from its given key
// return nil if the query parameter was not present
// calls `ctx.AbortWithError` if an error occurred while converting the query parameter value into a boolean
func GetQueryParamAsBool(ctx *gin.Context, key string) (*bool, error) {
	value := ctx.Query(key)
	if value == "" {
		return nil, nil
	}
	boolValue, err := strconv.ParseBool(ctx.Param(key))
	if err != nil {
		ctx.AbortWithError(http.StatusBadRequest, errors.NewBadParameterError(fmt.Sprintf("query parameter '%s' is not a valid boolean", key), value))
		return nil, err
	}
	return &boolValue, err
}
