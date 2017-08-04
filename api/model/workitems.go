package model

import (
	"fmt"

	"github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/google/jsonapi"
)

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
