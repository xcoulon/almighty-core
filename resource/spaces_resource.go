package resource

import (
	"net/http"

	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/resource/model"
	"github.com/gin-gonic/gin"
	"github.com/google/jsonapi"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// SpacesResource the resource for spaces
type SpacesResource struct {
	db application.DB
}

// NewSpacesResource returns a new SpacesResource
func NewSpacesResource(db application.DB) SpacesResource {
	return SpacesResource{
		db: db,
	}
}

//GetByID gets a space resource by its ID
func (r SpacesResource) GetByID(ctx *gin.Context) {
	spaceID, err := uuid.FromString(ctx.Param("spaceID"))
	if err != nil {
		ctx.AbortWithError(http.StatusBadRequest, errs.Wrapf(err, "the space ID is not a valid UUID"))
	}
	s, err := r.db.Spaces().Load(ctx, spaceID)
	if err != nil {
		//TODO: retrieve the correct HTTP status for the given err
		ctx.AbortWithError(http.StatusInternalServerError, errs.Wrapf(err, "error while fetching the space with id=%s", spaceID.String()))
	}
	// convert the business-domain 'space' into a jsonapi-model space
	result := model.Space{
		ID:          s.ID.String(),
		Name:        s.Name,
		Description: s.Description,
		BackLogSize: 10,
	}

	// marshall the result into a JSON-API compliant doc
	ctx.Status(http.StatusOK)
	ctx.Header("Content-Type", jsonapi.MediaType)
	if err := jsonapi.MarshalPayload(ctx.Writer, &result); err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, errs.Wrapf(err, "error while fetching the space with id=%s", spaceID.String()))
	}
}
