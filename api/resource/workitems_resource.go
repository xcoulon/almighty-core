package resource

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/fabric8-services/fabric8-wit/api/model"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/criteria"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/log"
	query "github.com/fabric8-services/fabric8-wit/query/simple"
	"github.com/fabric8-services/fabric8-wit/rendering"
	"github.com/fabric8-services/fabric8-wit/search"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/gin-gonic/gin"
	"github.com/google/jsonapi"
	errs "github.com/pkg/errors"
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

// WorkItemsResource the resource for work items
type WorkItemsResource struct {
	db application.DB
}

// NewWorkItemsResource returns a new WorkItemsResource
func NewWorkItemsResource(db application.DB) WorkItemsResource {
	return WorkItemsResource{db: db}
}

//List lists the work items, given the query parameters passed in the context
func (r WorkItemsResource) List(ctx *gin.Context) {
	spaceID, err := uuid.FromString(ctx.Param("spaceID")) // the space ID param
	if err != nil {
		ctx.AbortWithError(401, errors.NewBadParameterError("space ID is not a valid UUID", err))
	}
	var additionalQuery []string
	filter := GetQueryParamAsString(ctx, FilterQueryParam)
	exp, err := query.Parse(filter)
	if err != nil {
		ctx.AbortWithError(401, errors.NewBadParameterError("could not parse filter", err))
	}
	expressionFilter := GetQueryParamAsString(ctx, ExpressionFilterQueryParam)
	if expressionFilter != nil {
		q := *expressionFilter
		// Better approach would be to convert string to Query instance itself.
		// Then add new AND clause with spaceID as another child of input query
		// Then convert new Query object into simple string
		queryWithSpaceID := fmt.Sprintf(`?filter[expression]={"%s":[{"space": "%s" }, %s]}`, search.Q_AND, spaceID, q)
		searchURL := app.SearchHref() + queryWithSpaceID
		ctx.Header("Location", searchURL)
		ctx.Status(http.StatusTemporaryRedirect)
		return
	}
	assigneeFilter := GetQueryParamAsString(ctx, AssigneeFilterQueryParam)
	if assigneeFilter != nil {
		if *assigneeFilter == none {
			exp = criteria.And(exp, criteria.IsNull("system.assignees"))
			additionalQuery = append(additionalQuery, "filter[assignee]=none")
		} else {
			exp = criteria.And(exp, criteria.Equals(criteria.Field("system.assignees"), criteria.Literal([]string{*assigneeFilter})))
			additionalQuery = append(additionalQuery, "filter[assignee]="+*assigneeFilter)
		}
	}
	iterationFilter := GetQueryParamAsString(ctx, IterationFilterQueryParam)
	if iterationFilter != nil {
		exp = criteria.And(exp, criteria.Equals(criteria.Field(workitem.SystemIteration), criteria.Literal(string(*iterationFilter))))
		additionalQuery = append(additionalQuery, "filter[iteration]="+*iterationFilter)
		// Update filter by adding child iterations if any
		application.Transactional(r.db, func(tx application.Application) error {
			iterationUUID, errConversion := uuid.FromString(*iterationFilter)
			if errConversion != nil {
				ctx.AbortWithError(401, errors.NewBadParameterError("invalid iteration ID", errConversion))
			}
			childrens, err := tx.Iterations().LoadChildren(ctx, iterationUUID)
			if err != nil {
				ctx.AbortWithError(401, errors.NewBadParameterError("unable to fetch children", err))
			}
			for _, child := range childrens {
				childIDStr := child.ID.String()
				exp = criteria.Or(exp, criteria.Equals(criteria.Field(workitem.SystemIteration), criteria.Literal(childIDStr)))
				additionalQuery = append(additionalQuery, "filter[iteration]="+childIDStr)
			}
			return nil
		})
	}
	workitemTypeFilter, err := GetQueryParamAsUUID(ctx, WorkItemTypeFilterQueryParam)
	if err != nil {
		return // context was already aborted
	}
	if workitemTypeFilter != nil {
		exp = criteria.And(exp, criteria.Equals(criteria.Field("Type"), criteria.Literal([]uuid.UUID{*workitemTypeFilter})))
		additionalQuery = append(additionalQuery, "filter[workitemtype]="+workitemTypeFilter.String())
	}
	areaFilter := GetQueryParamAsString(ctx, AreaFilterQueryParam)
	if areaFilter != nil {
		exp = criteria.And(exp, criteria.Equals(criteria.Field(workitem.SystemArea), criteria.Literal(string(*areaFilter))))
		additionalQuery = append(additionalQuery, "filter[area]="+*areaFilter)
	}
	workitemStateFilter := GetQueryParamAsString(ctx, WorkItemStateFilterQueryParam)
	if workitemStateFilter != nil {
		exp = criteria.And(exp, criteria.Equals(criteria.Field(workitem.SystemState), criteria.Literal(string(*workitemStateFilter))))
		additionalQuery = append(additionalQuery, "filter[workitemstate]="+*workitemStateFilter)
	}
	parentExistsFilter, err := GetQueryParamAsBool(ctx, ParentExistsFilterQueryParam)
	if err != nil {
		return // context was already aborted
	}
	if parentExistsFilter != nil {
		// no need to build expression: it is taken care in wi.List call
		// we need additionalQuery to make sticky filters in URL links
		additionalQuery = append(additionalQuery, "filter[parentexists]="+strconv.FormatBool(*parentExistsFilter))
	}
	pageOffset, err := GetQueryParamAsInt(ctx, PageOffsetQueryParam)
	if err != nil {
		return // context was already aborted
	}
	pageLimit, err := GetQueryParamAsInt(ctx, PageLimitQueryParam)
	if err != nil {
		return // context was already aborted
	}
	workitems, totalCount, err := r.db.WorkItems().List(ctx, spaceID, exp, parentExistsFilter, pageOffset, pageLimit)
	if err != nil {
		ctx.AbortWithError(500, errors.NewBadParameterError("error listing work items", err))
	}
	// hasChildren := workItemIncludeHasChildren(tx, ctx)
	items := make([]*model.WorkItem, len(workitems)) // has to be an array of pointer
	for i, wi := range workitems {
		items[i] = &model.WorkItem{
			ID:          wi.ID.String(),
			SpaceID:     wi.SpaceID.String(),
			Title:       wi.Fields[workitem.SystemTitle].(string),
			Description: wi.Fields[workitem.SystemDescription].(rendering.MarkupContent).Content,
		}
	}
	// result := model.WorkItems{
	// 	SpaceID:    spaceID.String(),
	// 	WorkItems:  items,
	// 	TotalCount: totalCount,
	// }
	// setPagingLinks(response.Links, buildAbsoluteURL(ctx.RequestData), len(workitems), offset, limit, count, additionalQuery...)
	// addFilterLinks(response.Links, ctx.RequestData)
	log.Info(nil, nil, "Marshalling %d items", len(items))
	p, err := jsonapi.Marshal(items)
	payload, ok := p.(*jsonapi.ManyPayload)
	if !ok {
		ctx.AbortWithError(http.StatusInternalServerError, errs.Wrap(err, "error while preparing the response payload"))
	}
	log.Info(nil, nil, "Adding meta total-count")
	payload.Meta = &jsonapi.Meta{
		"total-count": totalCount,
	}
	ctx.Status(http.StatusOK)
	ctx.Header("Content-Type", jsonapi.MediaType)
	log.Info(nil, nil, "encoding payload")
	if err := json.NewEncoder(ctx.Writer).Encode(payload); err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, errs.Wrapf(err, "error while fetching the space with id=%s", spaceID.String()))
	}

}
