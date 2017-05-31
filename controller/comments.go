package controller

import (
	"context"
	"fmt"
	"html"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/comment"
	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/log"
	"github.com/almighty/almighty-core/login"
	"github.com/almighty/almighty-core/markup"
	"github.com/almighty/almighty-core/markup/rendering"
	"github.com/almighty/almighty-core/rest"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
)

// CommentsController implements the comments resource.
type CommentsController struct {
	*goa.Controller
	db     application.DB
	config CommentsControllerConfiguration
}

// CommentsControllerConfiguration the configuration for CommentsController
type CommentsControllerConfiguration interface {
	GetCacheControlComments() string
}

// NewCommentsController creates a comments controller.
func NewCommentsController(service *goa.Service, db application.DB, config CommentsControllerConfiguration) *CommentsController {
	return &CommentsController{
		Controller: service.NewController("CommentsController"),
		db:         db,
		config:     config,
	}
}

// Show runs the show action.
func (c *CommentsController) Show(ctx *app.ShowCommentsContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		cmt, err := appl.Comments().Load(ctx, ctx.CommentID)
		if err != nil {
			jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrUnauthorized(err.Error()))
			return ctx.NotFound(jerrors)
		}
		return ctx.ConditionalEntity(*cmt, c.config.GetCacheControlComments, func() error {
			res := &app.CommentSingle{}
			// This code should change if others type of parents than WI are allowed
			includeParentWorkItem, err := CommentIncludeParentWorkItem(ctx, appl, cmt)
			if err != nil {
				return errors.NewNotFoundError("comment parentID", cmt.ParentID)
			}
			res.Data = ConvertComment(
				ctx,
				ctx.RequestData,
				appl,
				*cmt,
				includeParentWorkItem)
			return ctx.OK(res)
		})
	})
}

// Update does PATCH comment
func (c *CommentsController) Update(ctx *app.UpdateCommentsContext) error {
	identityID, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
	}

	return application.Transactional(c.db, func(appl application.Application) error {
		cm, err := appl.Comments().Load(ctx.Context, ctx.CommentID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}

		if *identityID != cm.CreatedBy {
			// need to use the goa.NewErrorClass() func as there is no native support for 403 in goa
			// and it is not planned to be supported yet: https://github.com/goadesign/goa/pull/1030
			return jsonapi.JSONErrorResponse(ctx, goa.NewErrorClass("forbidden", 403)("User is not the comment author"))
		}

		cm.Body = *ctx.Payload.Data.Attributes.Body
		cm.Markup = markup.NilSafeGetMarkup(ctx.Payload.Data.Attributes.Markup)
		err = appl.Comments().Save(ctx.Context, cm, *identityID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}

		// This code should change if others type of parents than WI are allowed
		includeParentWorkItem, err := CommentIncludeParentWorkItem(ctx, appl, cm)
		if err != nil {
			return errors.NewNotFoundError("comment parentID", cm.ParentID)
		}

		res := &app.CommentSingle{
			Data: ConvertComment(ctx, ctx.RequestData, appl, *cm, includeParentWorkItem),
		}
		return ctx.OK(res)
	})
}

// Delete does DELETE comment
func (c *CommentsController) Delete(ctx *app.DeleteCommentsContext) error {
	identityID, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, goa.ErrUnauthorized(err.Error()))
	}

	return application.Transactional(c.db, func(appl application.Application) error {
		cm, err := appl.Comments().Load(ctx.Context, ctx.CommentID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		if *identityID != cm.CreatedBy {
			// need to use the goa.NewErrorClass() func as there is no native support for 403 in goa
			// and it is not planned to be supported yet: https://github.com/goadesign/goa/pull/1030
			return jsonapi.JSONErrorResponse(ctx, goa.NewErrorClass("forbidden", 403)("User is not the comment author"))
		}

		err = appl.Comments().Delete(ctx.Context, cm.ID, *identityID)
		if err != nil {
			return jsonapi.JSONErrorResponse(ctx, err)
		}
		return ctx.OK([]byte{})
	})
}

// CommentConvertFunc is a open ended function to add additional links/data/relations to a Comment during
// conversion from internal to API
type CommentConvertFunc func(*goa.RequestData, *comment.Comment, *app.Comment)

// ConvertComments converts between internal and external REST representation
func ConvertComments(ctx context.Context, request *goa.RequestData, appl application.Application, comments []comment.Comment, additional ...CommentConvertFunc) []*app.Comment {
	var result = []*app.Comment{}
	for _, cmt := range comments {
		result = append(result, ConvertComment(ctx, request, appl, cmt, additional...))
	}
	return result
}

// ConvertCommentsResourceID converts between internal and external REST representation, ResourceIdentificationObject only
func ConvertCommentsResourceID(request *goa.RequestData, comments []comment.Comment, additional ...CommentConvertFunc) []*app.Comment {
	var cs = []*app.Comment{}
	for _, c := range comments {
		cs = append(cs, ConvertCommentResourceID(request, c, additional...))
	}
	return cs
}

// ConvertCommentResourceID converts between internal and external REST representation, ResourceIdentificationObject only
func ConvertCommentResourceID(request *goa.RequestData, comment comment.Comment, additional ...CommentConvertFunc) *app.Comment {
	c := &app.Comment{
		Type: "comments",
		ID:   &comment.ID,
	}
	for _, add := range additional {
		add(request, &comment, c)
	}
	return c
}

// ConvertComment converts between internal and external REST representation
func ConvertComment(ctx context.Context, request *goa.RequestData, appl application.Application, cmt comment.Comment, additional ...CommentConvertFunc) *app.Comment {
	selfURL := rest.AbsoluteURL(request, app.CommentsHref(cmt.ID))
	markup := markup.NilSafeGetMarkup(&cmt.Markup)
	workitemRepo := appl.WorkItems()
	renderer := rendering.NewMarkupRenderer(appl.WorkItems(), rest.BaseURL(request))
	parentUUID, err := uuid.FromString(cmt.ParentID)
	if err != nil {
		// here we just log the error and return a nil 'rendered' body of the given comment
		log.Error(ctx, map[string]interface{}{"comment_id": cmt.ID}, "Failed to parse the comment's parent id with value '%s'", cmt.ParentID)
		return nil
	}
	wi, err := workitemRepo.LoadByID(ctx, parentUUID)
	if err != nil {
		// here we just log the error and return a nil 'rendered' body of the given comment
		log.Error(ctx, map[string]interface{}{"comment_id": cmt.ID}, "Failed to find the parent work item for the comment")
		return nil
	}
	bodyRendered, err := renderer.RenderMarkupToHTML(ctx, wi.SpaceID, html.EscapeString(cmt.Body), cmt.Markup)
	if err != nil {
		// here we just log the error and return a nil 'rendered' body of the given comment
		log.Error(ctx, map[string]interface{}{"comment_id": cmt.ID}, "Failed to render the body for the comment")
		return nil
	}
	relatedCreatorLink := rest.AbsoluteURL(request, fmt.Sprintf("%s/%s", usersEndpoint, cmt.CreatedBy.String()))
	c := &app.Comment{
		Type: "comments",
		ID:   &cmt.ID,
		Attributes: &app.CommentAttributes{
			Body:         &cmt.Body,
			BodyRendered: bodyRendered,
			Markup:       &markup,
			CreatedAt:    &cmt.CreatedAt,
			UpdatedAt:    &cmt.UpdatedAt,
		},
		Relationships: &app.CommentRelations{
			CreatedBy: &app.CommentCreatedBy{
				Data: &app.IdentityRelationData{
					Type: "identities",
					ID:   &cmt.CreatedBy,
				},
				Links: &app.GenericLinks{
					Related: &relatedCreatorLink,
				},
			},
		},
		Links: &app.GenericLinks{
			Self: &selfURL,
		},
	}
	for _, add := range additional {
		add(request, &cmt, c)
	}
	return c
}

// HrefFunc generic function to greate a relative Href to a resource
type HrefFunc func(id interface{}) string

// CommentIncludeParentWorkItem includes a "parent" relation to a WorkItem
func CommentIncludeParentWorkItem(ctx context.Context, appl application.Application, c *comment.Comment) (CommentConvertFunc, error) {
	// NOTE: This function assumes that the comment is bound to a WorkItem. Therefore,
	// we can extract the space out of this WI.
	wiID, err := uuid.FromString(c.ParentID)
	if err != nil {
		return nil, err
	}
	wi, err := appl.WorkItems().LoadByID(ctx, wiID)
	if err != nil {
		return nil, err
	}

	return func(request *goa.RequestData, comment *comment.Comment, data *app.Comment) {
		hrefFunc := func(obj interface{}) string {
			return fmt.Sprintf(app.WorkitemHref(wi.SpaceID, "%v"), obj)
		}
		CommentIncludeParent(request, comment, data, hrefFunc, APIStringTypeWorkItem)
	}, nil
}

// CommentIncludeParent adds the "parent" relationship to this Comment
func CommentIncludeParent(request *goa.RequestData, comment *comment.Comment, data *app.Comment, ref HrefFunc, parentType string) {
	parentSelf := rest.AbsoluteURL(request, ref(comment.ParentID))

	data.Relationships.Parent = &app.RelationGeneric{
		Data: &app.GenericData{
			Type: &parentType,
			ID:   &comment.ParentID,
		},
		Links: &app.GenericLinks{
			Self: &parentSelf,
		},
	}
}
