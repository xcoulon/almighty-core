package model

import (
	"fmt"

	"github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/google/jsonapi"
)

//Space the Space type of resource to (un)marshall in the JSON-API requests/responses
type Space struct {
	ID          string `jsonapi:"primary,spaces"`
	Version     int    `jsonapi:"attr,version"`
	Name        string `jsonapi:"attr,name"`
	Description string `jsonapi:"attr,description"`
}

func (s Space) JSONAPILinks() *jsonapi.Links {
	config := configuration.Get()
	return &jsonapi.Links{
		"self": jsonapi.Link{
			Href: fmt.Sprintf("%[1]s/api/spaces/%[2]s", config.GetAPIServiceURL(), s.ID),
		},
		"workitems": jsonapi.Link{
			Href: fmt.Sprintf("%[1]s/api/spaces/%[2]s", config.GetAPIServiceURL(), s.ID),
			Meta: map[string]interface{}{
				"counts": map[string]uint{
					"likes": 4,
				},
			},
		},
	}
}

// Invoked for each relationship defined on the Space struct when marshaled
// func (s Space) JSONAPIRelationshipLinks(relation string) *jsonapi.Links {
// 	if relation == "workitems" {
// 		return &jsonapi.Links{
// 			"related": fmt.Sprintf("https://example.com/spaces/%s/workitems", s.ID),
// 		}
// 	}
// 	return nil
// }
