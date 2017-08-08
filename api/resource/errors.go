package resource

import (
	"net/http"
	"strconv"

	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/gin-gonic/gin"
	"github.com/google/jsonapi"
	errs "github.com/pkg/errors"
)

// abortWithError aborts the context with the given error
func abortWithError(ctx *gin.Context, err error) {
	status := getHTTPStatus(err)
	log.Error(ctx, map[string]interface{}{"status": status, "error": err.Error()}, "Aborting context after error occurred")
	ctx.Status(status)
	ctx.Header("Content-Type", jsonapi.MediaType)
	jsonapi.MarshalErrors(ctx.Writer, []*jsonapi.ErrorObject{{
		Status: strconv.Itoa(status),
		Meta:   &map[string]interface{}{"error": err.Error()},
	}})
}

// getHTTPStatus gets the HTTP response status for the given error
func getHTTPStatus(err error) int {
	switch err := err.(type) {
	case errors.BadParameterError:
		return http.StatusBadRequest
	case errors.NotFoundError:
		return http.StatusNotFound
	default:
		// see if the underlying cause error was wrapped
		cause := errs.Cause(err)
		if cause != nil {
			return getHTTPStatus(cause)
		}
		return http.StatusInternalServerError
	}
}
