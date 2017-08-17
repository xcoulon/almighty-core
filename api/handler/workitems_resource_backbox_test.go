package handler_test

import (
	"bytes"
	"context"
	"net/http"
	"testing"

	"github.com/fabric8-services/fabric8-wit/api/model"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/google/jsonapi"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type WorkItemsResourceBlackBoxTestSuite struct {
	gormtestsupport.DBTestSuite
	clean func()
	repo  workitem.WorkItemTypeRepository
	ctx   context.Context
}

func TestWorkItemsResource(t *testing.T) {
	// resource.Require(t, resource.Database)
	suite.Run(t, &WorkItemsResourceBlackBoxTestSuite{DBTestSuite: gormtestsupport.NewDBTestSuite("../../config.yaml")})
}

func (s *WorkItemsResourceBlackBoxTestSuite) TestListWorkItemsOK() {
	r, err := http.NewRequest(http.MethodGet, "/api/spaces/2e0698d8-753e-4cef-bb7c-f027634824a2/workitems", nil)
	require.Nil(s.T(), err)
	verify(s.DBTestSuite, r, http.StatusOK)
}

func (s *WorkItemsResourceBlackBoxTestSuite) TestShowWorkItemOK() {
	r, err := http.NewRequest(http.MethodGet, "/api/workitems/c870914b-7942-4b87-8271-3afda49004e0", nil)
	require.Nil(s.T(), err)
	verify(s.DBTestSuite, r, http.StatusOK)
}

func (s *WorkItemsResourceBlackBoxTestSuite) TestCreateWorkItemOK() {
	testIdentity := createOneRandomUserIdentity(context.Background(), s.DB)
	title := "A title"
	description := "A description"
	state := workitem.SystemStateNew
	wi := model.WorkItem{
		Title:       &title,
		Description: &description,
		State:       &state,
		Type: &model.WorkItemType{
			ID: "26787039-b68f-4e28-8814-c2f93be1ef4e",
		},
	}
	payload := bytes.NewBuffer(make([]byte, 0))
	err := jsonapi.MarshalPayload(payload, &wi)
	require.Nil(s.T(), err)
	r, err := http.NewRequest(http.MethodPost, "/api/spaces/2e0698d8-753e-4cef-bb7c-f027634824a2/workitems", payload)
	require.Nil(s.T(), err) // generate/sign an auth token
	r.Header.Set("Authorization", "Bearer "+makeTokenString("HS256", testIdentity.ID.String()))
	verify(s.DBTestSuite, r, http.StatusCreated)
}

func (s *WorkItemsResourceBlackBoxTestSuite) TestCreateWorkItemKOMissingJWT() {
	title := "A title"
	description := "A description"
	state := workitem.SystemStateNew
	wi := model.WorkItem{
		Title:       &title,
		Description: &description,
		State:       &state,
		Type: &model.WorkItemType{
			ID: "26787039-b68f-4e28-8814-c2f93be1ef4e",
		},
	}
	payload := bytes.NewBuffer(make([]byte, 0))
	err := jsonapi.MarshalPayload(payload, &wi)
	require.Nil(s.T(), err)
	r, err := http.NewRequest(http.MethodPost, "/api/spaces/2e0698d8-753e-4cef-bb7c-f027634824a2/workitems", payload)
	require.Nil(s.T(), err)
	verify(s.DBTestSuite, r, http.StatusUnauthorized)
}

func (s *WorkItemsResourceBlackBoxTestSuite) TestCreateWorkItemKOInvalidCredentials() {
	title := "A title"
	description := "A description"
	state := workitem.SystemStateNew
	wi := model.WorkItem{
		Title:       &title,
		Description: &description,
		State:       &state,
		Type: &model.WorkItemType{
			ID: "26787039-b68f-4e28-8814-c2f93be1ef4e",
		},
	}
	payload := bytes.NewBuffer(make([]byte, 0))
	err := jsonapi.MarshalPayload(payload, &wi)
	require.Nil(s.T(), err)
	r, err := http.NewRequest(http.MethodPost, "/api/spaces/2e0698d8-753e-4cef-bb7c-f027634824a2/workitems", payload)
	require.Nil(s.T(), err)
	// generate/sign an auth token
	r.Header.Set("Authorization", "Bearer "+makeTokenString("HS256", "foo"))
	verify(s.DBTestSuite, r, http.StatusForbidden)
}
