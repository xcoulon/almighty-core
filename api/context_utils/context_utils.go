package context

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/gin-gonic/gin"
	"github.com/google/jsonapi"
	errs "github.com/pkg/errors"
)

func BaseURL(req *http.Request) string {
	url := req.URL
	return fmt.Sprintf("%[1]s://%[2]s", url.Scheme, url.Host)
}

// OK Responds with a '200 OK' response
func OK(ctx *gin.Context, result interface{}) {
	ctx.Status(http.StatusOK)
	ctx.Header("Content-Type", jsonapi.MediaType)
	log.Info(ctx, map[string]interface{}{"payload": result}, "encoding payload of type %T", result)
	switch result := result.(type) {
	case *jsonapi.ManyPayload:
		log.Info(ctx, map[string]interface{}{"payload": result}, "encoding many-payload")
		if err := json.NewEncoder(ctx.Writer).Encode(result); err != nil {
			AbortWithError(ctx, err)
		}
	default:
		if err := jsonapi.MarshalPayload(ctx, ctx.Writer, result); err != nil {
			AbortWithError(ctx, err)
		}
	}
}

// NotModified Responds with a '304 NotModified' response
func NotModified(ctx *gin.Context) {
	ctx.Status(http.StatusNotModified)
}

// Created Responds with a '201 Created' response with response body and a 'Location' header
func Created(ctx *gin.Context, result interface{}, location string) {
	ctx.Status(http.StatusCreated)
	ctx.Header("Content-Type", jsonapi.MediaType)
	ctx.Header("Location", location)
	if err := jsonapi.MarshalPayload(ctx, ctx.Writer, result); err != nil {
		AbortWithError(ctx, err)
	}
}

// AbortWithError aborts the context with the given error
func AbortWithError(ctx *gin.Context, err error) {
	status := getHTTPStatus(err)
	log.Error(ctx, map[string]interface{}{"status": status, "error": err.Error()}, "Aborting context after error occurred")
	ctx.Header("Content-Type", jsonapi.MediaType)
	jsonResponse := bytes.NewBuffer(make([]byte, 0))
	jsonapi.MarshalErrors(jsonResponse, []*jsonapi.ErrorObject{{
		Status: strconv.Itoa(status),
		Meta:   &map[string]interface{}{"error": err.Error()},
	}})
	ctx.AbortWithStatusJSON(status, jsonResponse.String())
}

// getHTTPStatus gets the HTTP response status for the given error
func getHTTPStatus(err error) int {
	log.Error(nil, map[string]interface{}{"error": err.Error(), "cause": errs.Cause(err)}, "Getting HTTP status after error occurred")

	switch err := err.(type) {
	case errors.BadParameterError:
		return http.StatusBadRequest
	case errors.NotFoundError:
		return http.StatusNotFound
	case errors.UnauthorizedError:
		return http.StatusUnauthorized
	case errors.ForbiddenError:
		return http.StatusForbidden
	default:
		// see if the underlying cause error was wrapped
		cause := errs.Cause(err)
		if cause != nil && cause != err {
			return getHTTPStatus(cause)
		}
		return http.StatusInternalServerError
	}
}
