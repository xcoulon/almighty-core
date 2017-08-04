package resource

import (
	"net/http"

	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/gin-gonic/gin"
	"github.com/google/jsonapi"
	errs "github.com/pkg/errors"
)

// abortWithError aborts the context with the given error
func abortWithError(ctx *gin.Context, err error) {
	var status int
	switch err := err.(type) {
	case errors.BadParameterError:
		status = http.StatusBadRequest
	case errors.NotFoundError:
		status = http.StatusNotFound
	default:
		// see if the underlying cause error was wrapped
		cause := errs.Cause(err)
		if cause != nil {
			abortWithError(ctx, cause)
		}
		status = http.StatusInternalServerError
	}
	ctx.Status(status)
	ctx.Header("Content-Type", jsonapi.MediaType)
	jsonapi.MarshalErrors(ctx.Writer, []*jsonapi.ErrorObject{{
		Status: "400",
		Meta:   &map[string]interface{}{"error": err.Error()},
	}})
}
