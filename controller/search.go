package controller

import (
	"fmt"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/log"
	"github.com/almighty/almighty-core/search"
	"github.com/almighty/almighty-core/space"

	"github.com/goadesign/goa"
	errs "github.com/pkg/errors"
)

type searchConfiguration interface {
	GetHTTPAddress() string
}

// SearchController implements the search resource.
type SearchController struct {
	*goa.Controller
	db            application.DB
	configuration searchConfiguration
}

// NewSearchController creates a search controller.
func NewSearchController(service *goa.Service, db application.DB, configuration searchConfiguration) *SearchController {
	if db == nil {
		panic("db must not be nil")
	}
	return &SearchController{Controller: service.NewController("SearchController"), db: db, configuration: configuration}
}

// Show runs the show action.
func (c *SearchController) Show(ctx *app.ShowSearchContext) error {
	var offset int
	var limit int

	offset, limit = computePagingLimits(ctx.PageOffset, ctx.PageLimit)

	// ToDo : Keep URL registeration central somehow.
	hostString := ctx.RequestData.Host
	if hostString == "" {
		hostString = c.configuration.GetHTTPAddress()
	}
	urlRegexString := fmt.Sprintf("(?P<domain>%s)(?P<path>/work-item/list/detail/)(?P<id>\\d*)", hostString)
	search.RegisterAsKnownURL(search.HostRegistrationKeyForListWI, urlRegexString)
	urlRegexString = fmt.Sprintf("(?P<domain>%s)(?P<path>/work-item/board/detail/)(?P<id>\\d*)", hostString)
	search.RegisterAsKnownURL(search.HostRegistrationKeyForBoardWI, urlRegexString)

	return application.Transactional(c.db, func(appl application.Application) error {
		//return transaction.Do(c.ts, func() error {
		result, c, err := appl.SearchItems().SearchFullText(ctx.Context, ctx.Q, &offset, &limit, ctx.SpaceID)
		count := int(c)
		if err != nil {
			cause := errs.Cause(err)
			switch cause.(type) {
			case errors.BadParameterError:
				jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrBadRequest(fmt.Sprintf("error listing work items: %s", err.Error())))
				return ctx.BadRequest(jerrors)
			default:
				log.Error(ctx, map[string]interface{}{
					"err": err,
				}, "unable to list the work items")
				jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrInternal(err.Error()))
				return ctx.InternalServerError(jerrors)
			}
		}

		response := app.SearchWorkItemList{
			Links: &app.PagingLinks{},
			Meta:  &app.WorkItemListResponseMeta{TotalCount: count},
			Data:  ConvertWorkItems(ctx, ctx.RequestData, appl, result),
		}

		setPagingLinks(response.Links, buildAbsoluteURL(ctx.RequestData), len(result), offset, limit, count, "q="+ctx.Q)
		return ctx.OK(&response)
	})
}

// Spaces runs the space search action.
func (c *SearchController) Spaces(ctx *app.SpacesSearchContext) error {
	q := ctx.Q
	if q == "" {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrBadRequest(fmt.Errorf("empty search query not allowed")))
	} else if q == "*" {
		q = "" // Allow empty query if * specified
	}

	var result []space.Space
	var count int
	var err error

	offset, limit := computePagingLimits(ctx.PageOffset, ctx.PageLimit)

	return application.Transactional(c.db, func(appl application.Application) error {
		var resultCount uint64
		result, resultCount, err = appl.Spaces().Search(ctx, &q, &offset, &limit)
		count = int(resultCount)
		if err != nil {
			cause := errs.Cause(err)
			switch cause.(type) {
			case errors.BadParameterError:
				return jsonapi.JSONErrorResponse(ctx, goa.ErrBadRequest(fmt.Sprintf("error listing spaces: %s", err.Error())))
			default:
				log.Error(ctx, map[string]interface{}{
					"query":  q,
					"offset": offset,
					"limit":  limit,
					"err":    err,
				}, "unable to list spaces")
				return jsonapi.JSONErrorResponse(ctx, goa.ErrInternal(err.Error()))
			}
		}

		spaceData, err := ConvertSpacesFromModel(ctx.Context, c.db, ctx.RequestData, result)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		response := app.SearchSpaceList{
			Links: &app.PagingLinks{},
			Meta:  &app.SpaceListMeta{TotalCount: count},
			Data:  spaceData,
		}
		setPagingLinks(response.Links, buildAbsoluteURL(ctx.RequestData), len(result), offset, limit, count, "q="+q)

		return ctx.OK(&response)
	})
}

// Users runs the user search action.
func (c *SearchController) Users(ctx *app.UsersSearchContext) error {

	q := ctx.Q
	if q == "" {
		return ctx.BadRequest(goa.ErrBadRequest(fmt.Errorf("empty search query not allowed")))
	}

	var result []account.Identity
	var count int
	var err error

	offset, limit := computePagingLimits(ctx.PageOffset, ctx.PageLimit)

	err = application.Transactional(c.db, func(appl application.Application) error {
		result, count, err = appl.Identities().Search(ctx, q, offset, limit)
		return err
	})
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"err": err,
		}, "unable to run search query on users.")
		ctx.InternalServerError()
	}

	var users []*app.UserData
	for i := range result {
		ident := result[i]
		id := ident.ID.String()
		userID := ident.User.ID.String()
		users = append(users, &app.UserData{
			// FIXME : should be "users" in the long term
			Type: "identities",
			ID:   &id,
			Attributes: &app.UserDataAttributes{
				CreatedAt:  &ident.User.CreatedAt,
				UpdatedAt:  &ident.User.UpdatedAt,
				Username:   &ident.Username,
				FullName:   &ident.User.FullName,
				ImageURL:   &ident.User.ImageURL,
				Bio:        &ident.User.Bio,
				URL:        &ident.User.URL,
				UserID:     &userID,
				IdentityID: &id,
				Email:      &ident.User.Email,
				Company:    &ident.User.Company,
			},
		})
	}

	// If there are no search results ensure that the 'data' section of the jsonapi
	// response is not null, rather [] (empty array)
	if users == nil {
		users = []*app.UserData{}
	}
	response := app.UserList{
		Data:  users,
		Links: &app.PagingLinks{},
		Meta:  &app.UserListMeta{TotalCount: count},
	}
	setPagingLinks(response.Links, buildAbsoluteURL(ctx.RequestData), len(result), offset, limit, count, "q="+q)

	return ctx.OK(&response)
}
