package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/fabric8-services/fabric8-wit/api/authz"
	contextutils "github.com/fabric8-services/fabric8-wit/api/context_utils"
	"github.com/fabric8-services/fabric8-wit/api/model"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/area"
	"github.com/fabric8-services/fabric8-wit/auth"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/iteration"
	"github.com/fabric8-services/fabric8-wit/login"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/fabric8-services/fabric8-wit/space"
	"github.com/gin-gonic/gin"
	"github.com/google/jsonapi"
	errs "github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// SpacesResourceConfiguration is the config interface for SpacesResource
type SpacesResourceConfiguration interface {
	GetCacheControlWorkItems() string
	GetAPIServiceURL() string
}

// SpacesResource the resource for spaces
type SpacesResource struct {
	db              application.DB
	config          SpacesResourceConfiguration
	resourceManager *auth.AuthzResourceManager
}

// NewSpacesResource returns a new SpacesResource
func NewSpacesResource(db application.DB, config SpacesResourceConfiguration, resourceManager *auth.AuthzResourceManager) SpacesResource {
	return SpacesResource{
		db:              db,
		config:          config,
		resourceManager: resourceManager,
	}
}

type SpacesResourceCreateContext struct {
	*gin.Context
	Space model.Space `gin:"body"`
}

// OK Responds with a '200 OK' response
func (ctx *SpacesResourceCreateContext) OK(result interface{}) {
	OK(ctx.Context, result)
}

//NewSpacesResourceCreateContext initializes a new SpacesResourceCreateContext context from the 'gin' context
func NewSpacesResourceCreateContext(ctx *gin.Context) (*SpacesResourceCreateContext, error) {
	payload := model.Space{}
	err := jsonapi.UnmarshalPayload(ctx.Request.Body, &payload)
	if err != nil {
		return nil, errors.NewConversionError(err.Error())
	}
	return &SpacesResourceCreateContext{
		Context: ctx,
		Space:   payload,
	}, nil
}

// Create handles the space creation requests
func (r SpacesResource) Create(ctx *gin.Context) {
	createCtx, err := NewSpacesResourceCreateContext(ctx)
	if err != nil {
		contextutils.AbortWithError(ctx, err)
		return
	}
	currentUserID, _ := authz.GetUserID(ctx)
	if createCtx.Space.Name == "" {
		contextutils.AbortWithError(ctx, errors.NewBadParameterError("name", createCtx.Space.Name))
		return
	}
	spaceID := uuid.NewV4()
	if createCtx.Space.ID != "" {
		spaceID, err = uuid.FromString(createCtx.Space.ID)
		if err != nil {
			contextutils.AbortWithError(ctx, errors.NewBadParameterError("id", spaceID))
			return
		}
	}

	var createdSpace *space.Space
	err = application.Transactional(r.db, func(appl application.Application) error {
		newSpace := space.Space{
			ID:          spaceID,
			Name:        createCtx.Space.Name,
			Description: createCtx.Space.Description,
			OwnerId:     *currentUserID,
		}
		createdSpace, err = appl.Spaces().Create(ctx, &newSpace)
		if err != nil {
			return errs.Wrapf(err, "failed to create space named '%s'", newSpace.Name)
		}
		/*
			Should we create the new area
			- over the wire(service) something like app.NewCreateSpaceAreasContext(..), OR
			- as part of a db transaction ?

			The argument 'for' creating it at a transaction level is :
			You absolutely need both space creation + area creation
			to happen in a single transaction as per requirements.
		*/

		newArea := area.Area{
			ID:      uuid.NewV4(),
			SpaceID: createdSpace.ID,
			Name:    createdSpace.Name,
		}
		err = appl.Areas().Create(ctx, &newArea)
		if err != nil {
			return errs.Wrapf(err, "failed to create area for space named '%s'", createdSpace.Name)
		}

		// Similar to above, we create a root iteration for this new space
		userActive := false
		newIteration := iteration.Iteration{
			ID:         uuid.NewV4(),
			SpaceID:    createdSpace.ID,
			Name:       createdSpace.Name,
			UserActive: &userActive,
		}
		err = appl.Iterations().Create(ctx, &newIteration)
		if err != nil {
			return errs.Wrapf(err, "failed to create iteration for space named '%s'", createdSpace.Name)
		}

		kcSpaceResource, err := r.resourceManager.CreateSpace(ctx, ctx.Request, spaceID.String())
		if err != nil {
			return errs.Wrapf(err, "failed to create KC resource for space with name '%s'", createdSpace.Name)
		}
		// finally, create the `space resource` using the remote KC data
		spaceResource := &space.Resource{
			ResourceID:   kcSpaceResource.Data.ResourceID,
			PolicyID:     kcSpaceResource.Data.PolicyID,
			PermissionID: kcSpaceResource.Data.PermissionID,
			SpaceID:      spaceID,
		}
		_, err = appl.SpaceResources().Create(ctx, spaceResource)
		return err
	})
	if err != nil {
		contextutils.AbortWithError(ctx, err)
		return
	}

	// convert the business-domain 'space' into a jsonapi-model space
	result := model.Space{
		ID:          createdSpace.ID.String(),
		Name:        createdSpace.Name,
		Description: createdSpace.Description,
		BackLogSize: 10,
	}

	// marshall the result into a JSON-API compliant response
	ctx.Status(http.StatusCreated)
	ctx.Header("Content-Type", jsonapi.MediaType)
	location := rest.AbsoluteURL(ctx.Request, fmt.Sprintf("%[1]s/api/spaces/%[2]s", r.config.GetAPIServiceURL(), createdSpace.ID.String()))
	ctx.Header("Location", location)
	if err := jsonapi.MarshalPayload(ctx.Writer, &result); err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, errs.Wrapf(err, "error while writing response after creating space with id '%s'", spaceID.String()))
		return
	}
}

//Show shows a space resource by its ID
func (r SpacesResource) Show(ctx *gin.Context) {
	_, err := login.ContextIdentity(ctx)
	if err != nil {
		ctx.AbortWithError(http.StatusUnauthorized, err)
		return
	}
	spaceID, err := uuid.FromString(ctx.Param("spaceID"))
	if err != nil {
		contextutils.AbortWithError(ctx, errs.Wrapf(err, "the space ID is not a valid UUID"))
		return
	}
	s, err := r.db.Spaces().Load(ctx, spaceID)
	if err != nil {
		//TODO: retrieve the correct HTTP status for the given err
		contextutils.AbortWithError(ctx, errs.Wrapf(err, "error while fetching the space with id=%s", spaceID.String()))
	}
	// convert the business-domain 'space' into a jsonapi-model space
	result := model.Space{
		ID:          s.ID.String(),
		Name:        s.Name,
		Description: s.Description,
		BackLogSize: 10,
	}

	// marshall the result into a JSON-API compliant response
	ctx.Status(http.StatusOK)
	ctx.Header("Content-Type", jsonapi.MediaType)
	if err := jsonapi.MarshalPayload(ctx.Writer, &result); err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, errs.Wrapf(err, "error while fetching the space with id=%s", spaceID.String()))
		return
	}
}

// List runs the list action.
func (r SpacesResource) List(ctx *gin.Context) {
	_, err := login.ContextIdentity(ctx)
	if err != nil {
		ctx.AbortWithError(http.StatusUnauthorized, err)
	}
	pageNumber := GetQueryParamAsString(ctx, "page[number]")
	pageLimit, err := GetQueryParamAsInt(ctx, "page[limit]")
	offset, limit := computePagingLimits(pageNumber, pageLimit)
	spaces, cnt, err := r.db.Spaces().List(ctx, &offset, &limit)
	results := []*model.Space{}
	for _, s := range spaces {
		results = append(results, &model.Space{
			ID:          s.ID.String(),
			Name:        s.Name,
			Description: s.Description,
			BackLogSize: 10,
		})
	}
	p, err := jsonapi.Marshal(results)
	payload, ok := p.(*jsonapi.ManyPayload)
	if !ok {
		ctx.AbortWithError(http.StatusInternalServerError, errs.Wrap(err, "error while preparing the response payload"))
	}
	payload.Links = &jsonapi.Links{}
	first, prev, next, last := getPagingLinks(payload.Links, fmt.Sprintf("%[1]s/api/spaces/", r.config.GetAPIServiceURL()), int(cnt), offset, limit, 10, "")
	payload.Meta = &jsonapi.Meta{
		"total-count": cnt,
	}
	payload.Links = &jsonapi.Links{
		"first": first,
		"prev":  prev,
		"next":  next,
		"last":  last,
	}

	ctx.Status(http.StatusOK)
	ctx.Header("Content-Type", jsonapi.MediaType)
	if err := json.NewEncoder(ctx.Writer).Encode(payload); err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, errs.Wrapf(err, "Error while fetching the spaces"))
		return
	}
}
