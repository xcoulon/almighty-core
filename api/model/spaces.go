package model

import (
	"fmt"

	"github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/google/jsonapi"
)

//Space the Space type of resource to (un)marshall in the JSON-API requests/responses
type Space struct {
	ID          string     `jsonapi:"primary,spaces"`
	Name        string     `jsonapi:"attr,name"`
	Description string     `jsonapi:"attr,description"`
	BackLog     []WorkItem `jsonapi:"relation,backlog"`
	BackLogSize int        // carried in this struct to be exposed as the `meta/count` attribute in the `workitems`` links
}

// JSONAPILinks returns the links to the space
func (s Space) JSONAPILinks() *jsonapi.Links {
	config := configuration.LoadDefault()
	return &jsonapi.Links{
		"self": jsonapi.Link{
			Href: fmt.Sprintf("%[1]s/api/spaces/%[2]s", config.GetAPIServiceURL(), s.ID),
		},
	}
}

//JSONAPIRelationshipLinks is invoked for each relationship defined on the Space struct when marshaled
func (s Space) JSONAPIRelationshipLinks(relation string) *jsonapi.Links {
	config := configuration.LoadDefault()
	if relation == "backlog" {
		return &jsonapi.Links{
			"related": jsonapi.Link{
				Href: fmt.Sprintf("%[1]s/api/spaces/%[2]s/backlog", config.GetAPIServiceURL(), s.ID),
			},
		}
	}
	return nil
}

// JSONAPIRelationshipMeta Invoked for each relationship defined on the Post struct when marshaled
func (s Space) JSONAPIRelationshipMeta(relation string) *jsonapi.Meta {
	if relation == "backlog" {
		return &jsonapi.Meta{
			"totalCount": s.BackLogSize,
		}
	}
	return nil
}
