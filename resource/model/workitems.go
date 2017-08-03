package model

import (
	"fmt"

	"github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/google/jsonapi"
)

//WorkItems an array of work items
type WorkItems struct {
	SpaceID    string
	WorkItems  []WorkItem
	TotalCount int // for the meta/totalCount
}

// JSONAPILinks returns the links to the list of workitems
func (w WorkItems) JSONAPILinks() *jsonapi.Links {
	config := configuration.Get()
	return &jsonapi.Links{
		"self": jsonapi.Link{
			Href: fmt.Sprintf("%[1]s/api/spaces/%[2]s/workitems", config.GetAPIServiceURL(), w.SpaceID),
		},
	}
}

// //JSONAPIRelationshipLinks is invoked for each relationship defined on the Space struct when marshaled
// func (w WorkItems) JSONAPIRelationshipLinks(relation string) *jsonapi.Links {
// 	return nil
// }

// JSONAPIMeta is used to include document meta in response data
func (w WorkItems) JSONAPIMeta() *jsonapi.Meta {
	return &jsonapi.Meta{
		"totalCount": w.TotalCount,
	}
}

//WorkItem the WorkItem type of resource to (un)marshall in the JSON-API requests/responses
type WorkItem struct {
	ID          string `jsonapi:"primary,workitems"`
	SpaceID     string
	Title       string `jsonapi:"attr,title"`
	Description string `jsonapi:"attr,description"`
}

// JSONAPILinks returns the links to the work item
func (w WorkItem) JSONAPILinks() *jsonapi.Links {
	config := configuration.Get()
	return &jsonapi.Links{
		"self": jsonapi.Link{
			Href: fmt.Sprintf("%[1]s/api/spaces/%[2]s/workitems/%[3]s", config.GetAPIServiceURL(), w.SpaceID, w.ID),
		},
	}
}

//JSONAPIRelationshipLinks is invoked for each relationship defined on the Space struct when marshaled
func (w WorkItem) JSONAPIRelationshipLinks(relation string) *jsonapi.Links {
	return nil
}

// JSONAPIRelationshipMeta Invoked for each relationship defined on the Post struct when marshaled
func (w WorkItem) JSONAPIRelationshipMeta(relation string) *jsonapi.Meta {
	return nil
}
