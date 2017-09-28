package model

import (
	"context"
	"fmt"

	"github.com/fabric8-services/fabric8-wit/space"
	"github.com/gin-gonic/gin"
	"github.com/google/jsonapi"
	uuid "github.com/satori/go.uuid"
)

// ***********************************************
// Space
// ***********************************************

//Space the Space type of resource to (un)marshall in the JSON-API requests/responses
type Space struct {
	ID          *uuid.UUID    `jsonapi:"primary,spaces"`
	CreatedAt   *string       `jsonapi:"attr,created-at"`
	UpdatedAt   *string       `jsonapi:"attr,updated-at"`
	Name        *string       `jsonapi:"attr,name"`
	Description *string       `jsonapi:"attr,description"`
	Version     *int          `jsonapi:"attr,version"`
	BackLog     *SpaceBackLog `jsonapi:"relation,backlog"`
	Owner       *RelatedUser  `jsonapi:"relation,owned-by"`
	Iterations  []Iteration   `jsonapi:"relation,iterations"`
	// BackLogSize int               // carried in this struct to be exposed as the `meta/count` attribute in the `workitems`` links
}

// ConvertSpaceToModel converts the given space `s` to a `model.Space`
func ConvertSpaceToModel(ctx *gin.Context, s space.Space, backlogCount uint64) *Space {
	return &Space{
		ID:          &s.ID,
		CreatedAt:   formatRFC3339(s.CreatedAt),
		UpdatedAt:   formatRFC3339(s.UpdatedAt),
		Version:     &s.Version,
		Name:        &s.Name,
		Description: &s.Description,
		Owner:       &RelatedUser{ID: s.OwnerId},
		BackLog: &SpaceBackLog{
			TotalCount: backlogCount,
		},
	}
}

// ***********************************************
// Spaces
// ***********************************************

// Spaces the structure for multiple spaces
type Spaces struct {
	Data       []*Space
	TotalCount int
}

// ConvertSpacesToModel converts a slice of spaces
func ConvertSpacesToModel(ctx *gin.Context, spaces []space.Space) []*Space {
	result := make([]*Space, len(spaces))
	for i, s := range spaces {
		result[i] = ConvertSpaceToModel(ctx, s, 0)
	}
	return result
}

// JSONAPILinks returns the links to the space
func (s Space) JSONAPILinks(ctx context.Context) *jsonapi.Links {
	return &jsonapi.Links{
		"self": jsonapi.Link{
			Href: fmt.Sprintf("http://localhost:8080/api/spaces/%[1]s", s.ID),
		},
	}
}

//JSONAPIRelationshipLinks is invoked for each relationship defined on the Space struct when marshaled
func (s Space) JSONAPIRelationshipLinks(ctx context.Context, relation string) *jsonapi.Links {
	if relation == "backlog" && s.BackLog != nil {
		return &jsonapi.Links{
			"related": jsonapi.Link{
				Href: fmt.Sprintf("http://localhost:8080/api/spaces/%[1]s/backlog", s.ID),
				Meta: jsonapi.Meta{"totalCount": s.BackLog.TotalCount},
			},
		}
	} else if relation == "owned-by" && s.Owner != nil {
		return &jsonapi.Links{
			"related": jsonapi.Link{Href: fmt.Sprintf("http://localhost:8080/api/users/%[1]s", s.Owner.ID.String())},
		}
	} else if relation == "iterations" && s.Owner != nil {
		return &jsonapi.Links{
			"related": jsonapi.Link{Href: fmt.Sprintf("http://localhost:8080/api/spaces/%[1]s/iterations", s.ID.String())},
		}
	}
	return nil
}

// // JSONAPIRelationshipMeta Invoked for each relationship defined on the Post struct when marshaled
// func (s Space) JSONAPIRelationshipMeta(ctx context.Context, relation string) *jsonapi.Meta {
// 	if relation == "backlog" {
// 		return &jsonapi.Meta{
// 			"totalCount": s.BackLog.TotalCount,
// 		}
// 	}
// 	return nil
// }

// ***********************************************
// Work items backlog
// ***********************************************

// SpaceBackLog the relationship struct for a space's backlog
type SpaceBackLog struct {
	ID         *uuid.UUID `jsonapi:"primary,backlog"`
	TotalCount uint64
}
