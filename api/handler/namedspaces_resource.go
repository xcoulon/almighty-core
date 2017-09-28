package handler

import (
	"context"
	"fmt"

	contextutils "github.com/fabric8-services/fabric8-wit/api/context_utils"
	"github.com/fabric8-services/fabric8-wit/api/model"
	"github.com/fabric8-services/fabric8-wit/space"
	"github.com/google/jsonapi"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/gin-gonic/gin"
	errs "github.com/pkg/errors"
)

// NamedspacesResource implements the namedspaces handler.
type NamedspacesResource struct {
	db application.DB
}

// NewNamedspacesResource returns a new NamedspacesResource
func NewNamedspacesResource(db application.DB) NamedspacesResource {
	return NamedspacesResource{
		db: db,
	}
}

// *************************************************************************
// Show Namedspaces: handler for `GET /api/namedspaces/:userName/:spaceName`
// *************************************************************************

// ShowNamedspacesContext the context for the Show endpoint
type ShowNamedspacesContext struct {
	*gin.Context
	UserName  string `gin:"param,userName"`
	SpaceName string `gin:"param,spaceName"`
}

// NewShowNamedspacesContext initializes a new ShowNamedspacesContext from the given `ctx` context
func NewShowNamedspacesContext(ctx *gin.Context) (*ShowNamedspacesContext, error) {
	userName := contextutils.GetParamAsString(ctx, "userName")
	if userName == nil {
		return nil, errors.NewBadParameterError("userName", nil)
	}
	spaceName := contextutils.GetParamAsString(ctx, "spaceName")
	if spaceName == nil {
		return nil, errors.NewBadParameterError("spaceName", nil)
	}
	return &ShowNamedspacesContext{
		Context:   ctx,
		UserName:  *userName,
		SpaceName: *spaceName,
	}, nil
}

// OK Responds with a '200 OK' response
func (ctx *ShowNamedspacesContext) OK(result *model.Space) {
	contextutils.OK(ctx.Context, result)
}

// Show processes the `GET /api/:userName/:spaceName` requests
func (r *NamedspacesResource) Show(ctx *gin.Context) {
	showCtx, err := NewShowNamedspacesContext(ctx)
	if err != nil {
		contextutils.AbortWithError(ctx, err)
		return
	}
	var s *space.Space
	err = application.Transactional(r.db, func(appl application.Application) error {
		identity, err := loadKeyCloakIdentityByUserName(ctx, appl, showCtx.UserName)
		if err != nil {
			return err
		}
		s, err = appl.Spaces().LoadByOwnerAndName(showCtx, &identity.ID, &showCtx.SpaceName)
		return err
	})
	if err != nil {
		contextutils.AbortWithError(ctx, err)
		return
	}
	modelSpace := model.ConvertSpaceToModel(ctx, *s, 0)
	showCtx.OK(modelSpace)
}

// **************************************************************
// List Namedspaces: handler for `GET /api/namedspaces/:userName`
// **************************************************************

// ListNamedspacesContext the context for the Show endpoint
type ListNamedspacesContext struct {
	*gin.Context
	UserName   string `gin:"param,userName"`
	PageOffset int    `gin:"param,page[offset]"`
	PageLimit  int    `gin:"param,page[limit]"`
}

// NewListNamedspacesContext initializes a new ListNamedspacesContext from the given `ctx` context
func NewListNamedspacesContext(ctx *gin.Context) (*ListNamedspacesContext, error) {
	userName, err := contextutils.GetRequiredParamAsString(ctx, "userName")
	if err != nil {
		return nil, err
	}
	pageOffset, err := contextutils.GetQueryParamAsInt(ctx, "page[offset]")
	if err != nil {
		return nil, err
	}
	pageLimit, err := contextutils.GetQueryParamAsInt(ctx, "page[limit]")
	if err != nil {
		return nil, err
	}
	offset, limit := computePagingLimits(pageOffset, pageLimit)
	return &ListNamedspacesContext{
		Context:    ctx,
		UserName:   *userName,
		PageOffset: offset,
		PageLimit:  limit,
	}, nil
}

// OK Responds with a '200 OK' response
func (ctx *ListNamedspacesContext) OK(entities []*model.Space, totalCount int) {
	log.Info(ctx.Context, map[string]interface{}{
		"base_url":       fmt.Sprintf("%[1]s/api/spaces/", contextutils.BaseURL(ctx.Request)),
		"entities_count": len(entities),
		"total_count":    totalCount,
		"page_offset":    ctx.PageOffset,
		"page_limit":     ctx.PageLimit},
		"responding with entities")
	p, err := jsonapi.Marshal(ctx, entities)
	if err != nil {
		contextutils.AbortWithError(ctx.Context, err)
		return
	}
	payload, ok := p.(*jsonapi.ManyPayload)
	if !ok {
		contextutils.AbortWithError(ctx.Context, errs.Errorf("Payload is not of the expected type: %T", p))
		return
	}
	payload.Links = getPaginationLinks(fmt.Sprintf("%[1]s/api/spaces/", contextutils.BaseURL(ctx.Request)), len(entities), ctx.PageOffset, ctx.PageLimit, totalCount)
	payload.Meta = &jsonapi.Meta{
		"total-count": totalCount,
	}
	contextutils.OK(ctx.Context, payload)
}

// List processes the `GET /api/:userName` requests
func (r *NamedspacesResource) List(ctx *gin.Context) {
	listCtx, err := NewListNamedspacesContext(ctx)
	if err != nil {
		contextutils.AbortWithError(ctx, err)
		return
	}
	var spaces []space.Space
	var totalCount uint64
	err = application.Transactional(r.db, func(appl application.Application) error {
		identity, err := loadKeyCloakIdentityByUserName(listCtx, appl, listCtx.UserName)
		if err != nil {
			return err
		}
		spaces, totalCount, err = appl.Spaces().LoadByOwner(listCtx.Context, &identity.ID, &listCtx.PageOffset, &listCtx.PageLimit)
		return err
	})
	if err != nil {
		contextutils.AbortWithError(ctx, err)
		return
	}
	// convert domain objects
	modelSpaces := model.ConvertSpacesToModel(ctx, spaces)
	listCtx.OK(modelSpaces, int(totalCount))
}

func loadKeyCloakIdentityByUserName(ctx context.Context, appl application.Application, username string) (*account.Identity, error) {
	identities, err := appl.Identities().Query(account.IdentityFilterByUsername(username))
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"username": username,
		}, "Fail to locate identity for user")
		return nil, err
	}
	for _, identity := range identities {
		if identity.ProviderType == account.KeycloakIDP {
			return &identity, nil
		}
	}
	log.Error(ctx, map[string]interface{}{
		"username": username,
	}, "failed to locate Keycloak identity for user")
	return nil, errors.NewNotFoundError("identities", username)
}
