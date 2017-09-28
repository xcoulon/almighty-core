package handler

import (
	"fmt"

	apiauth "github.com/fabric8-services/fabric8-wit/api/auth"
	contextutils "github.com/fabric8-services/fabric8-wit/api/context_utils"
	"github.com/fabric8-services/fabric8-wit/api/model"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/area"
	"github.com/fabric8-services/fabric8-wit/auth"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/iteration"
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

// *********************************************************************
// Create space: handler for `POST /api/spaces`
// *********************************************************************

// CreateSpaceContext the context for space creation
type CreateSpaceContext struct {
	*gin.Context
	Space model.Space `gin:"body"`
}

//NewCreateSpaceContext initializes a new CreateSpaceContext context from the 'gin' context
func NewCreateSpaceContext(ctx *gin.Context) (*CreateSpaceContext, error) {
	payload := model.Space{}
	err := jsonapi.UnmarshalPayload(ctx.Request.Body, &payload)
	if err != nil {
		return nil, errors.NewConversionError(err.Error())
	}
	return &CreateSpaceContext{
		Context: ctx,
		Space:   payload,
	}, nil
}

// Created Responds with a '201 Created' response
func (ctx *CreateSpaceContext) Created(result *model.Space) {
	location := rest.AbsoluteURL(ctx.Request, fmt.Sprintf("%[1]s/api/spaces/%[2]s", contextutils.BaseURL(ctx.Request), result.ID.String()))
	contextutils.Created(ctx.Context, result, location)
}

// Create handles the space creation requests
func (r SpacesResource) Create(ctx *gin.Context) {
	createCtx, err := NewCreateSpaceContext(ctx)
	if err != nil {
		contextutils.AbortWithError(ctx, err)
		return
	}
	currentUserID, _ := apiauth.GetUserID(ctx)
	if createCtx.Space.Name == nil {
		contextutils.AbortWithError(ctx, errors.NewBadParameterError("name", createCtx.Space.Name))
		return
	}
	spaceID := uuid.NewV4()
	if createCtx.Space.ID != nil {
		spaceID = *createCtx.Space.ID
	}

	var createdSpace *space.Space
	err = application.Transactional(r.db, func(appl application.Application) error {
		newSpace := space.Space{
			ID:          spaceID,
			Name:        *createCtx.Space.Name,
			Description: *createCtx.Space.Description,
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
		newIteration := iteration.Iteration{
			ID:         uuid.NewV4(),
			SpaceID:    createdSpace.ID,
			Name:       createdSpace.Name,
			UserActive: false,
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
	modelSpace := model.ConvertSpaceToModel(ctx, *createdSpace, 0)
	// marshall the result into a JSON-API compliant response
	createCtx.Created(modelSpace)
}

// *********************************************************************
// Show space: handler for `GET /api/spaces/:spaceID`
// *********************************************************************

// ShowSpaceContext the context for showing a space
type ShowSpaceContext struct {
	*gin.Context
	SpaceID uuid.UUID `gin:"param,spaceID"`
}

//NewShowSpaceContext initializes a new ShowSpaceContext context from the 'gin' context
func NewShowSpaceContext(ctx *gin.Context) (*ShowSpaceContext, error) {
	spaceID, err := contextutils.GetParamAsUUID(ctx, "spaceID")
	if err != nil {
		return nil, err
	}
	return &ShowSpaceContext{
		Context: ctx,
		SpaceID: *spaceID,
	}, nil
}

// OK Responds with a '200 OK' response
func (ctx *ShowSpaceContext) OK(result *model.Space) {
	contextutils.OK(ctx.Context, result)
}

//Show shows a space by its ID
func (r SpacesResource) Show(ctx *gin.Context) {
	showCtx, err := NewShowSpaceContext(ctx)
	if err != nil {
		contextutils.AbortWithError(ctx, err)
		return
	}
	var s *space.Space
	err = application.Transactional(r.db, func(appl application.Application) error {
		var err error
		s, err = r.db.Spaces().Load(ctx, showCtx.SpaceID)
		return err
	})
	if err != nil {
		contextutils.AbortWithError(ctx, err)
		return
	}
	// convert the business-domain 'space' into a jsonapi-model space
	result := model.ConvertSpaceToModel(ctx, *s, 0)
	showCtx.OK(result)
}

// **************************************************************
// List Spaces: handler for `GET /api/spaces`
// **************************************************************

// ListSpacesContext the context for the `List spaces`` endpoint
type ListSpacesContext struct {
	*gin.Context
	PageOffset *int `gin:"param,page[offset]"`
	PageLimit  *int `gin:"param,page[limit]"`
}

// NewListSpacesContext initializes a new ListNamedspacesContext from the given `ctx` context
func NewListSpacesContext(ctx *gin.Context) (*ListSpacesContext, error) {
	pageOffset, err := contextutils.GetQueryParamAsInt(ctx, "page[offset]")
	if err != nil {
		return nil, err
	}
	pageLimit, err := contextutils.GetQueryParamAsInt(ctx, "page[limit]")
	if err != nil {
		return nil, err
	}
	return &ListSpacesContext{
		Context:    ctx,
		PageOffset: pageOffset,
		PageLimit:  pageLimit,
	}, nil
}

// OK Responds with a '200 OK' response
// func (ctx *ListSpacesContext) OK(entities model.Spaces) {
// 	p, err := jsonapi.Marshal(entities)
// 	if err != nil {
// 		contextutils.AbortWithError(ctx.Context, err)
// 		return
// 	}
// 	payload, _ := p.(*jsonapi.ManyPayload)
// 	payload.Links := getPaginationLinks(payload.Links, fmt.Sprintf("%[1]s/api/spaces/", contextutils.BaseURL(ctx.Request)), int(entities.TotalCount), offset, limit, 10, "")
// 	payload.Meta = &jsonapi.Meta{
// 		"total-count": entities.TotalCount,
// 	}

// 	ctx.Status(http.StatusOK)
// 	ctx.Header("Content-Type", jsonapi.MediaType)
// 	if err := json.NewEncoder(ctx.Writer).Encode(payload); err != nil {
// 		ctx.AbortWithError(http.StatusInternalServerError, errs.Wrapf(err, "Error while fetching the spaces"))
// 		return
// 	}
// 	contextutils.OK(ctx.Context, payload)
// }

// List processes the `GET /api/spaces` requests
func (r SpacesResource) List(ctx *gin.Context) {
	listCtx, err := NewListSpacesContext(ctx)
	if err != nil {
		contextutils.AbortWithError(ctx, err)
		return
	}
	offset, limit := computePagingLimits(listCtx.PageOffset, listCtx.PageLimit)
	spaces := []space.Space{}
	var totalCount uint64
	err = application.Transactional(r.db, func(appl application.Application) error {
		var err error
		spaces, totalCount, err = appl.Spaces().List(ctx, &offset, &limit)
		return err
	})
	if err != nil {
		contextutils.AbortWithError(ctx, err)
		return
	}
	modelSpaces := make([]*model.Space, len(spaces))
	for _, s := range spaces {
		modelSpaces = append(modelSpaces, model.ConvertSpaceToModel(ctx, s, totalCount))
	}
	// listCtx.OK(modelSpaces)
}
