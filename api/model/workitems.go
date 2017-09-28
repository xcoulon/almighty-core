package model

import (
	"context"
	"fmt"

	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/codebase"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/rendering"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/google/jsonapi"
	uuid "github.com/satori/go.uuid"
)

//WorkItem the WorkItem type of resource to (un)marshall in the JSON-API requests/responses
type WorkItem struct {
	ID                uuid.UUID     `jsonapi:"primary,workitems"`
	Version           int           `jsonapi:"attr,version"`
	Title             *string       `jsonapi:"attr,system.title"`
	State             *string       `jsonapi:"attr,system.state"`
	Description       *string       `jsonapi:"attr,system.description"`
	DescriptionMarkup *string       `jsonapi:"attr,system.description.markup"`
	RemoteItemID      *string       `jsonapi:"attr,system.remote_item_id"`
	Number            *int          `jsonapi:"attr,system.number"`
	Creator           *string       `jsonapi:"attr,system.creator"`
	CreatedAt         *string       `jsonapi:"attr,system.created_at"`
	UpdatedAt         *string       `jsonapi:"attr,system.updated_at"`
	Order             *string       `jsonapi:"attr,system.order"`
	Iteration         *string       `jsonapi:"attr,system.iteration"`
	Area              *string       `jsonapi:"attr,system.area"`
	CodeBase          *string       `jsonapi:"attr,system.codebase"`
	Type              *WorkItemType `jsonapi:"relation,baseType"`  // 'relation' fields must be pointers
	Assignees         *WorkItemType `jsonapi:"relation,assignees"` // 'relation' fields must be pointers
	SpaceID           string
}

// ConvertWorkItemToModel converts the given WorkItem to the given model representation
func ConvertWorkItemToModel(wi workitem.WorkItem) *WorkItem {
	title := wi.Fields[workitem.SystemTitle].(string)
	descriptionContent := wi.Fields[workitem.SystemDescription].(rendering.MarkupContent).Content
	descriptionMarkup := wi.Fields[workitem.SystemDescription].(rendering.MarkupContent).Markup
	return &WorkItem{
		ID:                wi.ID,
		SpaceID:           wi.SpaceID.String(),
		Version:           wi.Version,
		Title:             &title,
		Description:       &descriptionContent,
		DescriptionMarkup: &descriptionMarkup,
		Type:              &WorkItemType{ID: wi.Type.String()},
	}
}

// ConvertModelToWorkItem converts the received work item (in the JSON-API format) into a domain work item
func ConvertModelToWorkItem(ctx context.Context, appl application.Application, source WorkItem, target *workitem.WorkItem, spaceID uuid.UUID) error {
	log.Info(nil, nil, "Converting work item using: %+v", source)
	// construct default values from input WI
	target.Version = source.Version

	// FIXME: needs to retrieve assignees, area and iteration relationships, too

	if source.Type != nil {
		sourceType, err := uuid.FromString(source.Type.ID)
		if err != nil {
			return errors.NewBadParameterError("source.relationships.baseType.data.id", source.Type.ID)
		}
		target.Type = sourceType
	}

	if m := rendering.NewMarkupContentFromValue(source.Description); m != nil {
		// if no description existed before, set the new one
		if target.Fields[workitem.SystemDescription] == nil {
			target.Fields[workitem.SystemDescription] = *m
		} else {
			// only update the 'description' field in the existing description
			existingDescription := target.Fields[workitem.SystemDescription].(rendering.MarkupContent)
			existingDescription.Content = (*m).Content
			target.Fields[workitem.SystemDescription] = existingDescription
		}
	}
	if source.DescriptionMarkup != nil {
		if !rendering.IsMarkupSupported(*source.DescriptionMarkup) {
			return errors.NewBadParameterError("data.relationships.attributes[system.description].markup", *source.DescriptionMarkup)
		}
		if target.Fields[workitem.SystemDescription] == nil {
			target.Fields[workitem.SystemDescription] = rendering.MarkupContent{Markup: *source.DescriptionMarkup}
		} else {
			existingDescription := target.Fields[workitem.SystemDescription].(rendering.MarkupContent)
			existingDescription.Markup = *source.DescriptionMarkup
			target.Fields[workitem.SystemDescription] = existingDescription
		}
	}
	if source.CodeBase != nil {
		cb, err := codebase.NewCodebaseContentFromValue(*source.CodeBase)
		if err != nil {
			return err
		}
		setupCodebase(appl, cb, spaceID)
		target.Fields[workitem.SystemCodebase] = *source.CodeBase
	}
	// other fields are simply copied if they have been provided in the request
	if source.Title != nil {
		target.Fields[workitem.SystemTitle] = *source.Title
	}
	log.Info(nil, nil, "Converted work item: %+v", target)
	return nil
}

// setupCodebase is the link between CodebaseContent & Codebase
// setupCodebase creates a codebase and saves it's ID in CodebaseContent
// for future use
func setupCodebase(appl application.Application, cb *codebase.Content, spaceID uuid.UUID) error {
	if cb.CodebaseID == "" {
		defaultStackID := "java-centos"
		newCodeBase := codebase.Codebase{
			SpaceID: spaceID,
			Type:    "git",
			URL:     cb.Repository,
			StackID: &defaultStackID,
			//TODO: Think of making stackID dynamic value (from analyzer)
		}
		existingCB, err := appl.Codebases().LoadByRepo(context.Background(), spaceID, cb.Repository)
		if existingCB != nil {
			cb.CodebaseID = existingCB.ID.String()
			return nil
		}
		err = appl.Codebases().Create(context.Background(), &newCodeBase)
		if err != nil {
			return errors.NewInternalError(context.Background(), err)
		}
		cb.CodebaseID = newCodeBase.ID.String()
	}
	return nil
}

// JSONAPILinks returns the links to the work item
func (w WorkItem) JSONAPILinks(ctx context.Context) *jsonapi.Links {
	return &jsonapi.Links{
		"self": jsonapi.Link{
			Href: fmt.Sprintf("http://localhost:8080/api/workitems/%[1]s", w.ID),
		},
	}
}

//JSONAPIRelationshipLinks is invoked for each relationship defined on the WorkItem struct when marshalled
func (w WorkItem) JSONAPIRelationshipLinks(ctx context.Context, relation string) *jsonapi.Links {
	if relation == "baseType" {
		return &jsonapi.Links{
			"self": jsonapi.Link{
				Href: fmt.Sprintf("http://localhost:8080/api/workitemtypes/%[1]s", w.ID),
			},
		}
	}
	return nil
}

// JSONAPIRelationshipMeta Invoked for each relationship defined on the Post struct when marshaled
func (w WorkItem) JSONAPIRelationshipMeta(ctx context.Context, relation string) *jsonapi.Meta {
	return nil
}
