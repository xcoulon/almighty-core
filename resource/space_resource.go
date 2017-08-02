package resource

import (
	"net/http"

	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/model"
	"github.com/fabric8-services/fabric8-wit/space"
	"github.com/gin-gonic/gin"
	"github.com/google/jsonapi"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// SpaceResource for api2go routes.
type SpaceResource struct {
	SpaceRepository space.Repository
}

// NewSpaceResource returns a new SpaceResource
func NewSpaceResource(db application.DB) SpaceResource {
	spaceRepo := db.Spaces()
	return SpaceResource{
		SpaceRepository: spaceRepo,
	}
}

//GetByID gets a space resource by its ID
func (r SpaceResource) GetByID(ctx *gin.Context) {
	spaceID, err := uuid.FromString(ctx.Param("spaceID"))
	if err != nil {
		ctx.AbortWithError(http.StatusBadRequest, errs.Wrapf(err, "the space ID is not a valid UUID"))
	}
	s, err := r.SpaceRepository.Load(ctx, spaceID)
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
