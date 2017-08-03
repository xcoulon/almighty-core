package resource

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/gin-gonic/gin"
	uuid "github.com/satori/go.uuid"
)

// GetQueryParamAsString returns the query param as a string from its given key
// return nil if the query param was not present
func GetQueryParamAsString(ctx *gin.Context, key string) *string {
	value := ctx.Query(key)
	if value == "" {
		return nil
	}
	return &value
}

// GetQueryParamAsInt returns the query param as a integer from its given key
// return nil if the query param was not present
// calls `ctx.AbortWithError` if an error occurred while converting the query parameter value into an integer
func GetQueryParamAsInt(ctx *gin.Context, key string) (*int, error) {
	value := ctx.Query(key)
	if value == "" {
		return nil, nil
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		ctx.AbortWithError(http.StatusBadRequest, errors.NewBadParameterError(fmt.Sprintf("'%s' query param is not a valid integer", key), value))
	}
	return &intValue, err
}

// GetQueryParamAsUUID returns the query param as a int from its given key
// return nil if the query param was not present
// calls `ctx.AbortWithError` if an error occurred while converting the query parameter value into an UUID
func GetQueryParamAsUUID(ctx *gin.Context, key string) (*uuid.UUID, error) {
	value := ctx.Query(key)
	if value == "" {
		return nil, nil
	}
	uuidValue, err := uuid.FromString(ctx.Param(key))
	if err != nil {
		ctx.AbortWithError(http.StatusBadRequest, errors.NewBadParameterError(fmt.Sprintf("'%s' query param is not a valid UUID", key), value))
	}
	return &uuidValue, err
}

// GetQueryParamAsBool returns the query param as a boolean from its given key
// return nil if the query param was not present
// calls `ctx.AbortWithError` if an error occurred while converting the query parameter value into a boolean
func GetQueryParamAsBool(ctx *gin.Context, key string) (*bool, error) {
	value := ctx.Query(key)
	if value == "" {
		return nil, nil
	}
	boolValue, err := strconv.ParseBool(ctx.Param(key))
	if err != nil {
		ctx.AbortWithError(http.StatusBadRequest, errors.NewBadParameterError(fmt.Sprintf("'%s' query param is not a valid boolean", key), value))
	}
	return &boolValue, err
}
