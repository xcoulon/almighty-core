package resource_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fabric8-services/fabric8-wit/api"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/stretchr/testify/suite"
)

type WorkItemsResourceBlackBoxTestSuite struct {
	gormtestsupport.DBTestSuite
	clean func()
	repo  workitem.WorkItemTypeRepository
	ctx   context.Context
}

func TestWorkItemsResource(t *testing.T) {
	suite.Run(t, &WorkItemsResourceBlackBoxTestSuite{DBTestSuite: gormtestsupport.NewDBTestSuite("../../config.yaml")})
}

func (s *WorkItemsResourceBlackBoxTestSuite) TestListWorkItemsOK() {
	r, err := http.NewRequest(http.MethodGet, "/api/spaces/2e0698d8-753e-4cef-bb7c-f027634824a2", nil)
	if err != nil {
		s.T().Fatal(err)
	}
	// r.Header.Set(headerAccept, jsonapi.MediaType)

	rr := httptest.NewRecorder()
	httpEngine := api.NewGinEngine(gormapplication.NewGormDB(s.DB))
	httpEngine.ServeHTTP(rr, r)

	if e, a := http.StatusOK, rr.Code; e != a {
		s.T().Fatalf("Expected a status of %d, got %d", e, a)
	}
}
