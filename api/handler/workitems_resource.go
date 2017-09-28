package handler

import (
	"fmt"
	"net/http"
	"strconv"

	apiauth "github.com/fabric8-services/fabric8-wit/api/auth"
	contextutils "github.com/fabric8-services/fabric8-wit/api/context_utils"
	"github.com/fabric8-services/fabric8-wit/api/model"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/criteria"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/notification"
	query "github.com/fabric8-services/fabric8-wit/query/simple"
	"github.com/fabric8-services/fabric8-wit/rendering"
	"github.com/fabric8-services/fabric8-wit/rest"
	"github.com/fabric8-services/fabric8-wit/search"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/gin-gonic/gin"
	"github.com/google/jsonapi"
	uuid "github.com/satori/go.uuid"
)

const (
	FilterQueryParam              = "filter"                // a query language expression restricting the set of found work items
	ExpressionFilterQueryParam    = "filter[expression]"    // accepts query in JSON format and redirects to /api/search? API". Example: `{$AND: [{"space": "f73988a2-1916-4572-910b-2df23df4dcc3"}, {"state": "NEW"}]}`
	AssigneeFilterQueryParam      = "filter[assignee]"      // Work Items assigned to the given user
	IterationFilterQueryParam     = "filter[iteration]"     // IterationID to filter work items
	WorkItemTypeFilterQueryParam  = "filter[workitemtype]"  // ID of work item type to filter work items by
	WorkItemStateFilterQueryParam = "filter[workitemstate]" // work item state to filter work items by
	AreaFilterQueryParam          = "filter[area]"          // AreaID to filter work items
	ParentExistsFilterQueryParam  = "filter[parentexists]"  //if false list work items without any parent
	PageLimitQueryParam           = "page[limit]"           // Paging size
	PageOffsetQueryParam          = "page[offset]"          // Paging size
	none                          = "none"
)

// WorkItemControllerConfig the config interface for the WorkitemController
type WorkItemsResourceConfiguration interface {
	GetCacheControlWorkItems() string
}

// WorkItemsResource the resource for work items
type WorkItemsResource struct {
	db                  application.DB
	config              WorkItemsResourceConfiguration
	notificationChannel notification.Channel
}

// NewWorkItemsResource returns a new WorkItemsResource
func NewWorkItemsResource(db application.DB, notificationChannel notification.Channel, config WorkItemsResourceConfiguration) WorkItemsResource {
	return WorkItemsResource{
		db:                  db,
		notificationChannel: notificationChannel,
		config:              config,
	}
}

type WorkItemsResourceListContext struct {
	*gin.Context
	SpaceID             uuid.UUID `gin:"param,spaceID"`
	Filter              *string   `gin:"query,filter"`
	ExpressionFilter    *string
	AssigneeFilter      *string
	IterationFilter     *string
	WorkitemTypeFilter  *uuid.UUID
	AreaFilter          *string
	WorkitemStateFilter *string
	ParentExistsFilter  *bool
	PageOffset          *int
	PageLimit           *int
}

// Note: this kind of function could be generated, based on struct tags on WorkItemsResourceListContext fields.
func NewWorkItemsResourceListContext(ctx *gin.Context) (*WorkItemsResourceListContext, error) {
	spaceID, err := contextutils.GetParamAsUUID(ctx, "spaceID")
	if err != nil {
		return nil, err
	}
	filter := contextutils.GetQueryParamAsString(ctx, FilterQueryParam)
	expressionFilter := contextutils.GetQueryParamAsString(ctx, ExpressionFilterQueryParam)
	assigneeFilter := contextutils.GetQueryParamAsString(ctx, AssigneeFilterQueryParam)
	iterationFilter := contextutils.GetQueryParamAsString(ctx, IterationFilterQueryParam)
	workitemTypeFilter, err := contextutils.GetQueryParamAsUUID(ctx, WorkItemTypeFilterQueryParam)
	if err != nil {
		return nil, err
	}
	areaFilter := contextutils.GetQueryParamAsString(ctx, AreaFilterQueryParam)
	workitemStateFilter := contextutils.GetQueryParamAsString(ctx, WorkItemStateFilterQueryParam)
	parentExistsFilter, err := contextutils.GetQueryParamAsBool(ctx, ParentExistsFilterQueryParam)
	if err != nil {
		return nil, err
	}
	pageOffset, err := contextutils.GetQueryParamAsInt(ctx, PageOffsetQueryParam)
	if err != nil {
		return nil, err
	}
	pageLimit, err := contextutils.GetQueryParamAsInt(ctx, PageLimitQueryParam)
	if err != nil {
		return nil, err
	}
	return &WorkItemsResourceListContext{
		Context:             ctx,
		SpaceID:             *spaceID,
		Filter:              filter,
		ExpressionFilter:    expressionFilter,
		AssigneeFilter:      assigneeFilter,
		IterationFilter:     iterationFilter,
		WorkitemTypeFilter:  workitemTypeFilter,
		AreaFilter:          areaFilter,
		WorkitemStateFilter: workitemStateFilter,
		ParentExistsFilter:  parentExistsFilter,
		PageOffset:          pageOffset,
		PageLimit:           pageLimit,
	}, nil
}

// OK Responds with a '200 OK' response
func (ctx *WorkItemsResourceListContext) OK(result interface{}) {
	contextutils.OK(ctx.Context, result)
}

func (ctx *WorkItemsResourceListContext) NotModified() {
	contextutils.NotModified(ctx.Context)
}

//List lists the work items, given the query parameters passed in the request URI
func (r WorkItemsResource) List(ctx *gin.Context) {
	listCtx, err := NewWorkItemsResourceListContext(ctx)
	if err != nil {
		contextutils.AbortWithError(ctx, err)
		return
	}
	var additionalQuery []string
	exp, err := query.Parse(listCtx.Filter)
	if err != nil {
		contextutils.AbortWithError(ctx, errors.NewBadParameterError("filter", err))
		return
	}
	if listCtx.ExpressionFilter != nil {
		q := *listCtx.ExpressionFilter
		// Better approach would be to convert string to Query instance itself.
		// Then add new AND clause with spaceID as another child of input query
		// Then convert new Query object into simple string
		queryWithSpaceID := fmt.Sprintf(`?filter[expression]={"%s":[{"space": "%s" }, %s]}`, search.Q_AND, listCtx.SpaceID, q)
		searchURL := app.SearchHref() + queryWithSpaceID
		ctx.Header("Location", searchURL)
		ctx.Status(http.StatusTemporaryRedirect)
		return
	}
	if listCtx.AssigneeFilter != nil {
		if *listCtx.AssigneeFilter == none {
			exp = criteria.And(exp, criteria.IsNull("system.assignees"))
			additionalQuery = append(additionalQuery, "filter[assignee]=none")
		} else {
			exp = criteria.And(exp, criteria.Equals(criteria.Field("system.assignees"), criteria.Literal([]string{*listCtx.AssigneeFilter})))
			additionalQuery = append(additionalQuery, "filter[assignee]="+*listCtx.AssigneeFilter)
		}
	}
	if listCtx.IterationFilter != nil {
		exp = criteria.And(exp, criteria.Equals(criteria.Field(workitem.SystemIteration), criteria.Literal(string(*listCtx.IterationFilter))))
		additionalQuery = append(additionalQuery, "filter[iteration]="+*listCtx.IterationFilter)
		// Update filter by adding child iterations if any
		err := application.Transactional(r.db, func(tx application.Application) error {
			iterationUUID, err := uuid.FromString(*listCtx.IterationFilter)
			if err != nil {
				return errors.NewBadParameterError("iterationID", err)
			}
			childrens, err := tx.Iterations().LoadChildren(ctx, iterationUUID)
			if err != nil {
				return errors.NewInternalError(ctx, err)
			}
			for _, child := range childrens {
				childIDStr := child.ID.String()
				exp = criteria.Or(exp, criteria.Equals(criteria.Field(workitem.SystemIteration), criteria.Literal(childIDStr)))
				additionalQuery = append(additionalQuery, "filter[iteration]="+childIDStr)
			}
			return nil
		})
		if err != nil {
			contextutils.AbortWithError(ctx, err)
			return
		}
	}
	if listCtx.WorkitemTypeFilter != nil {
		exp = criteria.And(exp, criteria.Equals(criteria.Field("Type"), criteria.Literal([]uuid.UUID{*listCtx.WorkitemTypeFilter})))
		additionalQuery = append(additionalQuery, "filter[workitemtype]="+listCtx.WorkitemTypeFilter.String())
	}
	if listCtx.AreaFilter != nil {
		exp = criteria.And(exp, criteria.Equals(criteria.Field(workitem.SystemArea), criteria.Literal(string(*listCtx.AreaFilter))))
		additionalQuery = append(additionalQuery, "filter[area]="+*listCtx.AreaFilter)
	}
	if listCtx.WorkitemStateFilter != nil {
		exp = criteria.And(exp, criteria.Equals(criteria.Field(workitem.SystemState), criteria.Literal(string(*listCtx.WorkitemStateFilter))))
		additionalQuery = append(additionalQuery, "filter[workitemstate]="+*listCtx.WorkitemStateFilter)
	}
	if listCtx.ParentExistsFilter != nil {
		// no need to build expression: it is taken care in wi.List call
		// we need additionalQuery to make sticky filters in URL links
		additionalQuery = append(additionalQuery, "filter[parentexists]="+strconv.FormatBool(*listCtx.ParentExistsFilter))
	}
	workitems, totalCount, err := r.db.WorkItems().List(ctx, listCtx.SpaceID, exp, listCtx.ParentExistsFilter, listCtx.PageOffset, listCtx.PageLimit)
	if err != nil {
		contextutils.AbortWithError(ctx, err)
		return
	}
	listCtx.ConditionalEntities(workitems, r.config.GetCacheControlWorkItems, func() {
		// hasChildren := workItemIncludeHasChildren(tx, ctx)
		items := make([]*model.WorkItem, len(workitems)) // has to be an array of pointer
		for i, wi := range workitems {
			items[i] = model.ConvertWorkItemToModel(wi)
		}
		// setPagingLinks(response.Links, buildAbsoluteURL(ctx.RequestData), len(workitems), offset, limit, count, additionalQuery...)
		// addFilterLinks(response.Links, ctx.RequestData)
		p, err := jsonapi.Marshal(listCtx, items)
		payload, ok := p.(*jsonapi.ManyPayload)
		if !ok {
			contextutils.AbortWithError(ctx, err)
			return
		}
		payload.Meta = &jsonapi.Meta{
			"total-count": totalCount,
		}
		payload.Links = &jsonapi.Links{
			"self": jsonapi.Link{
				Href: fmt.Sprintf("http://localhost:8080/api/spaces/%[1]s/workitems", listCtx.SpaceID.String()),
			},
		}
		listCtx.OK(payload)
	})
}

//WorkItemsResourceShowContext the context to show a work item
type WorkItemsResourceShowContext struct {
	*gin.Context
	WorkitemID uuid.UUID `gin:"param,workitemID"`
}

// OK Responds with a '200 OK' response
func (ctx *WorkItemsResourceShowContext) OK(result interface{}) {
	contextutils.OK(ctx.Context, result)
}

//NewWorkItemsResourceShowContext initializes a new WorkItemsResourceShowContext context from the 'gin' context
func NewWorkItemsResourceShowContext(ctx *gin.Context) (*WorkItemsResourceShowContext, error) {
	workitemID, err := uuid.FromString(ctx.Param("workitemID")) // the workitem ID param
	if err != nil {
		return nil, errors.NewBadParameterError("workitemID", err)
	}
	return &WorkItemsResourceShowContext{
		Context:    ctx,
		WorkitemID: workitemID,
	}, nil
}

//Show shows a single work item, given the parameters passed in the request URI
func (r WorkItemsResource) Show(ctx *gin.Context) {
	showCtx, err := NewWorkItemsResourceShowContext(ctx)
	if err != nil {
		contextutils.AbortWithError(ctx, err)
		return
	}
	wi, err := r.db.WorkItems().LoadByID(ctx, showCtx.WorkitemID)
	if err != nil {
		log.Warn(ctx, nil, "Aborting with error: %s", err.Error())
		contextutils.AbortWithError(ctx, err)
		return
	}
	result := model.ConvertWorkItemToModel(*wi)
	log.Info(ctx, map[string]interface{}{"wi_id": result.ID, "type_id": result.Type.ID}, "Returning work item: %+v", result)
	showCtx.OK(result)
}

type WorkItemsResourceCreateContext struct {
	*gin.Context
	SpaceID  uuid.UUID      `gin:"param,spaceID"`
	WorkItem model.WorkItem `gin:"body"`
}

// Created Responds with a '201 Created' response
func (ctx *WorkItemsResourceCreateContext) Created(result interface{}, location string) {
	contextutils.Created(ctx.Context, result, location)
}

//NewWorkItemsResourceCreateContext initializes a new WorkItemsResourceCreateContext context from the 'gin' context
func NewWorkItemsResourceCreateContext(ctx *gin.Context) (*WorkItemsResourceCreateContext, error) {
	spaceID, err := contextutils.GetParamAsUUID(ctx, "spaceID")
	if err != nil {
		return nil, err
	}
	payloadItem := model.WorkItem{}
	if err := jsonapi.UnmarshalPayload(ctx.Request.Body, &payloadItem); err != nil {
		return nil, err
	}
	return &WorkItemsResourceCreateContext{
		Context:  ctx,
		SpaceID:  *spaceID,
		WorkItem: payloadItem,
	}, nil
}

//Create creates a new work item, given the JSON-API content passed in the request body
func (r WorkItemsResource) Create(ctx *gin.Context) {
	createCtx, err := NewWorkItemsResourceCreateContext(ctx)
	if err != nil {
		contextutils.AbortWithError(ctx, err)
		return
	}
	payloadWI := createCtx.WorkItem
	payloadDescription := rendering.NewMarkupContentFromLegacy(*payloadWI.Description)
	fields := make(map[string]interface{})
	fields[workitem.SystemTitle] = *payloadWI.Title
	fields[workitem.SystemDescription] = payloadDescription
	fields[workitem.SystemState] = *payloadWI.State
	if createCtx.WorkItem.Type == nil {
		contextutils.AbortWithError(ctx, errors.NewBadParameterError("type", err))
		return
	}
	wiType, err := uuid.FromString(createCtx.WorkItem.Type.ID)
	if err != nil {
		contextutils.AbortWithError(ctx, errors.NewBadParameterError("type", err))
		return
	}
	// creatorID, _ := uuid.FromString("e1e9b60a-0c8d-4450-83d3-b2dc44a8bc1c")
	creatorID, _ := apiauth.GetUserID(ctx)
	createdWI, err := r.db.WorkItems().Create(ctx, createCtx.SpaceID, wiType, fields, *creatorID)
	if err != nil {
		contextutils.AbortWithError(ctx, err)
		return
	}
	location := rest.AbsoluteURL(ctx.Request, fmt.Sprintf("/api/workitems/%[1]s", createdWI.ID))
	responseWI := model.ConvertWorkItemToModel(*createdWI)
	createCtx.Created(responseWI, location)
}

type WorkItemsResourceUpdateContext struct {
	*gin.Context
	WorkItemID uuid.UUID      `gin:"param,workitemID"`
	WorkItem   model.WorkItem `gin:"body"`
}

// OK Responds with a '200 OK' response
func (ctx *WorkItemsResourceUpdateContext) OK(result interface{}) {
	contextutils.OK(ctx.Context, result)
}

//NewWorkItemsResourceUpdateContext initializes a new WorkItemsResourceUpdateContext context from the 'gin' context
func NewWorkItemsResourceUpdateContext(ctx *gin.Context) (*WorkItemsResourceUpdateContext, error) {
	workitemID, err := uuid.FromString(ctx.Param("workitemID")) // the workitem ID param
	if err != nil {
		return nil, errors.NewBadParameterError("workitemID", err)
	}
	payload := model.WorkItem{}
	err = jsonapi.UnmarshalPayload(ctx.Request.Body, &payload)
	if err != nil {
		return nil, errors.NewConversionError(err.Error())
	}
	return &WorkItemsResourceUpdateContext{
		Context:    ctx,
		WorkItemID: workitemID,
		WorkItem:   payload,
	}, nil
}

//Update updates an existing work item, given the JSON-API content passed in the request body
func (r WorkItemsResource) Update(ctx *gin.Context) {
	updateCtx, err := NewWorkItemsResourceUpdateContext(ctx)
	if err != nil {
		contextutils.AbortWithError(ctx, err)
		return
	}
	currentUserID, _ := apiauth.GetUserID(ctx)
	wi, err := r.db.WorkItems().LoadByID(ctx, updateCtx.WorkItemID)
	if err != nil {
		contextutils.AbortWithError(ctx, err)
		return
	}
	err = application.Transactional(r.db, func(appl application.Application) error {
		// type with the old one after the WI has been converted.
		oldType := wi.Type
		err = model.ConvertModelToWorkItem(ctx, appl, updateCtx.WorkItem, wi, wi.SpaceID)
		if err != nil {
			return err
		}
		wi.Type = oldType
		wi, err = r.db.WorkItems().Save(ctx, wi.SpaceID, *wi, *currentUserID)
		if err != nil {
			return err
		}

		// hasChildren := workItemIncludeHasChildren(appl, ctx)
		// wi2 := ConvertWorkItem(ctx.RequestData, *wi, hasChildren)
		// resp := &app.WorkItemSingle{
		// 	Data: wi2,
		// 	Links: &app.WorkItemLinks{
		// 		Self: buildAbsoluteURL(ctx.RequestData),
		// 	},
		// }
		result := model.ConvertWorkItemToModel(*wi)
		// ctx.ResponseData.Header().Set("Last-Modified", lastModified(*wi))
		updateCtx.OK(result)
		return nil
	})
	if err != nil {
		contextutils.AbortWithError(ctx, err)
		return
	}
	if err == nil && r.notificationChannel != nil {
		r.notificationChannel.Send(ctx, notification.NewWorkItemUpdated(wi.ID.String()))
	}
}
