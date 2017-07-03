package controller

import (
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/jsonapi"
	"github.com/fabric8-services/fabric8-wit/login"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/fabric8-services/fabric8-wit/workitem/link"
	"github.com/goadesign/goa"
	//uuid "github.com/satori/go.uuid"
)

// WorkItemLinkCategoriesController implements the work-item-link-category resource.
type WorkItemLinkCategoriesController struct {
	*goa.Controller
	db application.DB
}

// NewWorkItemLinkCategoryController creates a WorkItemLinkCategoriesController.
func NewWorkItemLinkCategoryController(service *goa.Service, db application.DB) *WorkItemLinkCategoriesController {
	return &WorkItemLinkCategoriesController{
		Controller: service.NewController("WorkItemLinkCategoriesController"),
		db:         db,
	}
}

// enrichLinkCategorySingle includes related resources in the single's "included" array
func enrichLinkCategorySingle(ctx *workItemLinkContext, single app.WorkItemLinkCategorySingle) error {
	// Add "links" element
	selfURL := rest.AbsoluteURL(ctx.RequestData, ctx.LinkFunc(single.Data.ID))
	single.Data.Links = &app.GenericLinks{
		Self: &selfURL,
	}
	return nil
}

// enrichLinkCategoryList includes related resources in the list's "included" array
func enrichLinkCategoryList(ctx *workItemLinkContext, list *app.WorkItemLinkCategoryList) error {
	// Add "links" element
	for _, data := range list.Data {
		selfURL := rest.AbsoluteURL(ctx.RequestData, ctx.LinkFunc(*data.ID))
		data.Links = &app.GenericLinks{
			Self: &selfURL,
		}
	}
	return nil
}

// Create runs the create action.
func (c *WorkItemLinkCategoriesController) Create(ctx *app.CreateWorkItemLinkCategoryContext) error {
	currentUserIdentityID, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewUnauthorizedError(err.Error()))
	}
	return application.Transactional(c.db, func(appl application.Application) error {
		modelCategory := convertLinkCategoryToModel(app.WorkItemLinkCategorySingle{Data: ctx.Payload.Data})
		_, err := appl.WorkItemLinkCategories().Create(ctx.Context, &modelCategory)
		if err != nil {
			jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(err)
			return ctx.ResponseData.Service.Send(ctx.Context, httpStatusCode, jerrors)
		}
		appCategory := convertLinkCategoryFromModel(modelCategory)
		linkCtx := newWorkItemLinkContext(ctx.Context, appl, c.db, ctx.RequestData, ctx.ResponseData, app.WorkItemLinkCategoryHref, currentUserIdentityID)
		err = enrichLinkCategorySingle(linkCtx, appCategory)
		if err != nil {
			jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrInternal("Failed to enrich link category: %s", err.Error()))
			return ctx.InternalServerError(jerrors)
		}
		ctx.ResponseData.Header().Set("Location", app.WorkItemLinkCategoryHref(appCategory.Data.ID))
		return ctx.Created(&appCategory)
	})
}

// Show runs the show action.
func (c *WorkItemLinkCategoriesController) Show(ctx *app.ShowWorkItemLinkCategoryContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		modelCategory, err := appl.WorkItemLinkCategories().Load(ctx.Context, ctx.ID)
		if err != nil {
			jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(err)
			return ctx.ResponseData.Service.Send(ctx.Context, httpStatusCode, jerrors)
		}
		appCategory := convertLinkCategoryFromModel(*modelCategory)
		linkCtx := newWorkItemLinkContext(ctx.Context, appl, c.db, ctx.RequestData, ctx.ResponseData, app.WorkItemLinkCategoryHref, nil)
		err = enrichLinkCategorySingle(linkCtx, appCategory)
		if err != nil {
			jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrInternal("Failed to enrich link category: %s", err.Error()))
			return ctx.InternalServerError(jerrors)
		}
		return ctx.OK(&appCategory)
	})
}

// List runs the list action.
func (c *WorkItemLinkCategoriesController) List(ctx *app.ListWorkItemLinkCategoryContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		modelCategories, err := appl.WorkItemLinkCategories().List(ctx.Context)
		if err != nil {
			jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(err)
			return ctx.ResponseData.Service.Send(ctx.Context, httpStatusCode, jerrors)
		}
		// convert
		appCategories := app.WorkItemLinkCategoryList{}
		appCategories.Data = make([]*app.WorkItemLinkCategoryData, len(modelCategories))
		for index, value := range modelCategories {
			cat := convertLinkCategoryFromModel(value)
			appCategories.Data[index] = cat.Data
		}
		// TODO: When adding pagination, this must not be len(rows) but
		// the overall total number of elements from all pages.
		appCategories.Meta = &app.WorkItemLinkCategoryListMeta{
			TotalCount: len(modelCategories),
		}
		// Enrich
		linkCtx := newWorkItemLinkContext(ctx.Context, appl, c.db, ctx.RequestData, ctx.ResponseData, app.WorkItemLinkCategoryHref, nil)
		err = enrichLinkCategoryList(linkCtx, &appCategories)
		if err != nil {
			jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrInternal("Failed to enrich link categories: %s", err.Error()))
			return ctx.InternalServerError(jerrors)
		}
		return ctx.OK(&appCategories)
	})
}

// Delete runs the delete action.
func (c *WorkItemLinkCategoriesController) Delete(ctx *app.DeleteWorkItemLinkCategoryContext) error {
	return application.Transactional(c.db, func(appl application.Application) error {
		err := appl.WorkItemLinkCategories().Delete(ctx.Context, ctx.ID)
		if err != nil {
			jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(err)
			return ctx.ResponseData.Service.Send(ctx.Context, httpStatusCode, jerrors)
		}
		return ctx.OK([]byte{})
	})
}

// Update runs the update action.
func (c *WorkItemLinkCategoriesController) Update(ctx *app.UpdateWorkItemLinkCategoryContext) error {
	currentUserIdentityID, err := login.ContextIdentity(ctx)
	if err != nil {
		return jsonapi.JSONErrorResponse(ctx, errors.NewUnauthorizedError(err.Error()))
	}
	appCategory := app.WorkItemLinkCategorySingle{
		Data: ctx.Payload.Data,
	}
	if appCategory.Data.ID == nil {
		return errors.NewBadParameterError("data.id", appCategory.Data.ID)
	}
	if appCategory.Data.Attributes.Name == nil || *appCategory.Data.Attributes.Name == "" {
		return errors.NewBadParameterError("data.attributes.name", "nil or empty")
	}
	modelCategory := convertLinkCategoryToModel(appCategory)
	return application.Transactional(c.db, func(appl application.Application) error {
		savedModelCategory, err := appl.WorkItemLinkCategories().Save(ctx.Context, modelCategory)
		if err != nil {
			jerrors, httpStatusCode := jsonapi.ErrorToJSONAPIErrors(err)
			return ctx.ResponseData.Service.Send(ctx.Context, httpStatusCode, jerrors)
		}
		// convert to app representation
		savedAppCategory := convertLinkCategoryFromModel(*savedModelCategory)
		// Enrich
		linkCtx := newWorkItemLinkContext(ctx.Context, appl, c.db, ctx.RequestData, ctx.ResponseData, app.WorkItemLinkCategoryHref, currentUserIdentityID)
		err = enrichLinkCategorySingle(linkCtx, savedAppCategory)
		if err != nil {
			jerrors, _ := jsonapi.ErrorToJSONAPIErrors(goa.ErrInternal("Failed to enrich link category: %s", err.Error()))
			return ctx.InternalServerError(jerrors)
		}
		return ctx.OK(&savedAppCategory)
	})
}

// convertLinkCategoryFromModel converts work item link category from model to app representation
func convertLinkCategoryFromModel(t link.WorkItemLinkCategory) app.WorkItemLinkCategorySingle {
	var converted = app.WorkItemLinkCategorySingle{
		Data: &app.WorkItemLinkCategoryData{
			Type: link.EndpointWorkItemLinkCategories,
			ID:   &t.ID,
			Attributes: &app.WorkItemLinkCategoryAttributes{
				Name:        &t.Name,
				Description: t.Description,
				Version:     &t.Version,
			},
		},
	}
	return converted
}

// convertLinkCategoryToModel converts work item link category from app to app representation
func convertLinkCategoryToModel(t app.WorkItemLinkCategorySingle) link.WorkItemLinkCategory {
	var converted = link.WorkItemLinkCategory{}
	if t.Data.ID != nil {
		converted.ID = *t.Data.ID
	}
	if t.Data.Attributes.Version != nil {
		converted.Version = *t.Data.Attributes.Version
	}
	if t.Data.Attributes.Name != nil {
		converted.Name = *t.Data.Attributes.Name
	}
	if t.Data.Attributes.Description != nil {
		converted.Description = t.Data.Attributes.Description
	}
	return converted
}
