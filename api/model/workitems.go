package model

import (
	"fmt"

	"github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/fabric8-services/fabric8-wit/rendering"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/google/jsonapi"
)

//WorkItem the WorkItem type of resource to (un)marshall in the JSON-API requests/responses
type WorkItem struct {
	ID          string `jsonapi:"primary,workitems"`
	SpaceID     string
	Title       string        `jsonapi:"attr,title"`
	Description string        `jsonapi:"attr,description"`
	Type        *WorkItemType `jsonapi:"relation,baseType"` // 'relation' fields must be pointers
}

// NewWorkItem initializes a new WorkItem from the given model
func NewWorkItem(wi workitem.WorkItem) *WorkItem {
	return &WorkItem{
		ID:          wi.ID.String(),
		SpaceID:     wi.SpaceID.String(),
		Title:       wi.Fields[workitem.SystemTitle].(string),
		Description: wi.Fields[workitem.SystemDescription].(rendering.MarkupContent).Content,
		Type:        &WorkItemType{ID: wi.Type.String()},
	}
}

// JSONAPILinks returns the links to the work item
func (w WorkItem) JSONAPILinks() *jsonapi.Links {
	config := configuration.Get()
	return &jsonapi.Links{
		"self": jsonapi.Link{
			Href: fmt.Sprintf("%[1]s/api/workitems/%[2]s", config.GetAPIServiceURL(), w.ID),
		},
	}
}

//JSONAPIRelationshipLinks is invoked for each relationship defined on the WorkItem struct when marshalled
func (w WorkItem) JSONAPIRelationshipLinks(relation string) *jsonapi.Links {
	config := configuration.Get()
	if relation == "baseType" {
		return &jsonapi.Links{
			"self": jsonapi.Link{
				Href: fmt.Sprintf("%[1]s/api/workitemtypes/%[2]s", config.GetAPIServiceURL(), w.ID),
			},
		}
	}
	return nil
}

// JSONAPIRelationshipMeta Invoked for each relationship defined on the Post struct when marshaled
func (w WorkItem) JSONAPIRelationshipMeta(relation string) *jsonapi.Meta {
	return nil
}
