package controller

import (
	"fmt"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/application"
	"github.com/almighty/almighty-core/errors"
	"github.com/almighty/almighty-core/jsonapi"
	"github.com/almighty/almighty-core/markup"
	"github.com/almighty/almighty-core/markup/rendering"
	"github.com/almighty/almighty-core/rest"
	"github.com/goadesign/goa"
	uuid "github.com/satori/go.uuid"
)

const (
	RenderingType = "rendering"
	RenderedValue = "value"
)

// RenderController implements the render resource.
type RenderController struct {
	*goa.Controller
	db application.DB
}

// NewRenderController creates a render controller.
func NewRenderController(service *goa.Service, db application.DB) *RenderController {
	return &RenderController{
		Controller: service.NewController("RenderController"),
		db:         db,
	}
}

// Render runs the render action.
func (c *RenderController) Render(ctx *app.RenderRenderContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		contentValue := ctx.Payload.Data.Attributes.Content
		markupValue := ctx.Payload.Data.Attributes.Markup
		if !markup.IsMarkupSupported(markupValue) {
			return jsonapi.JSONErrorResponse(ctx, errors.NewBadParameterError("Unsupported markup type", markupValue))
		}
		renderer := rendering.NewMarkupRenderer(appl.WorkItems(), rest.BaseURL(ctx.RequestData))
		htmlResult, err := renderer.RenderMarkupToHTML(ctx, ctx.Payload.Data.Attributes.SpaceID, contentValue, markupValue)
		if err != nil {
			// rendering failed: let's assume this is an internal server error :(
			return jsonapi.JSONErrorResponse(ctx, errors.NewInternalError(fmt.Sprintf("Failed to render the given content with the given markup type: %s", err.Error())))
		}
		res := &app.MarkupRenderingSingle{Data: &app.MarkupRenderingData{
			ID:   uuid.NewV4().String(),
			Type: RenderingType,
			Attributes: &app.MarkupRenderingDataAttributes{
				RenderedContent: *htmlResult,
			}}}
		return ctx.OK(res)

	})
}
