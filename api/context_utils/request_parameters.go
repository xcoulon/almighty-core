package context

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/gin-gonic/gin"
	uuid "github.com/satori/go.uuid"
)

// GetParamAsString returns the query parameter as a string from its given key
// return nil if the request parameter was not present
func GetParamAsString(ctx *gin.Context, key string) *string {
	value := ctx.Param(key)
	if value == "" {
		return nil
	}
	return &value
}

// GetParamAsString returns the query parameter as a string from its given key
// return a BadParameter if the request parameter was not present
func GetRequiredParamAsString(ctx *gin.Context, key string) (*string, error) {
	value := ctx.Param(key)
	if value == "" {
		return nil, errors.NewBadParameterError(fmt.Sprintf("parameter '%s' is found in the request URI", key), nil)
	}
	return &value, nil
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
// returns an `errors.BadParameterError` if an error occurred while converting the query parameter value into an integer
func GetQueryParamAsInt(ctx *gin.Context, key string) (*int, error) {
	value := ctx.Query(key)
	if value == "" {
		return nil, nil
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return nil, errors.NewBadParameterError(fmt.Sprintf("query parameter '%s' is not a valid integer", key), value)
	}
	return &intValue, nil
}

// GetRequiredQueryParamAsInt returns the query parameter as a integer from its given key
// return nil if the query parameter was not present
// returns an `errors.BadParameterError` if an error occurred while converting the query parameter value into an integer,
// or if the query parameter was missing in the request
func GetRequiredQueryParamAsInt(ctx *gin.Context, key string) (*int, error) {
	value, err := GetQueryParamAsInt(ctx, key)
	if value == nil {
		return nil, errors.NewBadParameterError(fmt.Sprintf("query parameter '%s' is not found in the request URI", key), nil)
	}
	return value, err
}

// GetQueryParamAsUUID returns the query parameter as a int from its given key
// return nil if the query parameter was not present
// returns an `errors.BadParameterError` if an error occurred while converting the query parameter value into a UUID
func GetQueryParamAsUUID(ctx *gin.Context, key string) (*uuid.UUID, error) {
	value := ctx.Query(key)
	if value == "" {
		return nil, nil
	}
	uuidValue, err := uuid.FromString(ctx.Param(key))
	if err != nil {
		return nil, errors.NewBadParameterError(fmt.Sprintf("query parameter '%s' is not a valid integer", key), value)
	}
	return &uuidValue, nil
}

// GetQueryParamAsBool returns the query parameter as a boolean from its given key
// return nil if the query parameter was not present
// returns an `errors.BadParameterError` if an error occurred while converting the query parameter value into a boolean
func GetQueryParamAsBool(ctx *gin.Context, key string) (*bool, error) {
	value := ctx.Query(key)
	if value == "" {
		return nil, nil
	}
	boolValue, err := strconv.ParseBool(ctx.Param(key))
	if err != nil {
		return nil, errors.NewBadParameterError(fmt.Sprintf("query parameter '%s' is not a valid integer", key), value)
	}
	return &boolValue, nil
}
