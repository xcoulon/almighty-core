package resource_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/fabric8-services/fabric8-wit/api"
	"github.com/fabric8-services/fabric8-wit/api/model"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/google/jsonapi"
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
	if err != nil {
		s.T().Fatal(err)
	}
	// r.Header.Set(headerAccept, jsonapi.MediaType)

	rr := httptest.NewRecorder()
	httpEngine := api.NewGinEngine(gormapplication.NewGormDB(s.DB), s.Configuration)
	httpEngine.ServeHTTP(rr, r)

	if e, a := http.StatusOK, rr.Code; e != a {
		s.T().Fatalf("Expected a status of %d, got %d", e, a)
	}
}

func (s *WorkItemsResourceBlackBoxTestSuite) TestShowWorkItemOK() {
	r, err := http.NewRequest(http.MethodGet, "/api/workitems/c870914b-7942-4b87-8271-3afda49004e0", nil)
	if err != nil {
		s.T().Fatal(err)
	}
	// r.Header.Set(headerAccept, jsonapi.MediaType)

	rr := httptest.NewRecorder()
	httpEngine := api.NewGinEngine(gormapplication.NewGormDB(s.DB), s.Configuration)
	httpEngine.ServeHTTP(rr, r)

	if e, a := http.StatusOK, rr.Code; e != a {
		s.T().Fatalf("Expected a status of %d, got %d", e, a)
	}
}

func (s *WorkItemsResourceBlackBoxTestSuite) TestCreateWorkItemOK() {
	testIdentity := createOneRandomUserIdentity(context.Background(), s.DB)
	wi := model.WorkItem{
		Title:       "A title",
		Description: "A description",
		State:       workitem.SystemStateNew,
		Type: &model.WorkItemType{
			ID: "26787039-b68f-4e28-8814-c2f93be1ef4e",
		},
	}
	payload := bytes.NewBuffer(make([]byte, 0))
	if err := jsonapi.MarshalPayload(payload, &wi); err != nil {
		s.T().Fatal(err)
	}
	r, err := http.NewRequest(http.MethodPost, "/api/spaces/2e0698d8-753e-4cef-bb7c-f027634824a2/workitems", payload)
	if err != nil {
		s.T().Fatal(err)
	}
	// generate/sign an auth token
	r.Header.Set("Authorization", "Bearer "+makeTokenString("HS256", testIdentity.ID.String()))
	rr := httptest.NewRecorder()
	httpEngine := api.NewGinEngine(gormapplication.NewGormDB(s.DB), s.Configuration)
	httpEngine.ServeHTTP(rr, r)
	if e, a := http.StatusCreated, rr.Code; e != a {
		s.T().Logf("Response body: \n%v", rr.Body.String())
		s.T().Fatalf("Expected a status of %d, got %d", e, a)
	}
}

func (s *WorkItemsResourceBlackBoxTestSuite) TestCreateWorkItemKOMissingJWT() {
	wi := model.WorkItem{
		Title:       "A title",
		Description: "A description",
		State:       workitem.SystemStateNew,
		Type: &model.WorkItemType{
			ID: "26787039-b68f-4e28-8814-c2f93be1ef4e",
		},
	}
	payload := bytes.NewBuffer(make([]byte, 0))
	if err := jsonapi.MarshalPayload(payload, &wi); err != nil {
		s.T().Fatal(err)
	}
	r, err := http.NewRequest(http.MethodPost, "/api/spaces/2e0698d8-753e-4cef-bb7c-f027634824a2/workitems", payload)
	if err != nil {
		s.T().Fatal(err)
	}

	rr := httptest.NewRecorder()
	httpEngine := api.NewGinEngine(gormapplication.NewGormDB(s.DB), s.Configuration)
	httpEngine.ServeHTTP(rr, r)
	// s.T().Logf("Response body: \n%v", rr.Body.String())
	if e, a := http.StatusUnauthorized, rr.Code; e != a {
		s.T().Logf("Response body: \n%v", rr.Body.String())
		s.T().Fatalf("Expected a status of %d, got %d", e, a)
	}
}

func (s *WorkItemsResourceBlackBoxTestSuite) TestCreateWorkItemKOInvalidCredentials() {
	wi := model.WorkItem{
		Title:       "A title",
		Description: "A description",
		State:       workitem.SystemStateNew,
		Type: &model.WorkItemType{
			ID: "26787039-b68f-4e28-8814-c2f93be1ef4e",
		},
	}
	payload := bytes.NewBuffer(make([]byte, 0))
	if err := jsonapi.MarshalPayload(payload, &wi); err != nil {
		s.T().Fatal(err)
	}
	r, err := http.NewRequest(http.MethodPost, "/api/spaces/2e0698d8-753e-4cef-bb7c-f027634824a2/workitems", payload)
	if err != nil {
		s.T().Fatal(err)
	}
	// generate/sign an auth token
	r.Header.Set("Authorization", "Bearer "+makeTokenString("HS256", "foo"))

	rr := httptest.NewRecorder()
	httpEngine := api.NewGinEngine(gormapplication.NewGormDB(s.DB), s.Configuration)
	httpEngine.ServeHTTP(rr, r)
	// s.T().Logf("Response body: \n%v", rr.Body.String())
	if e, a := http.StatusForbidden, rr.Code; e != a {
		s.T().Logf("Response body: \n%v", rr.Body.String())
		s.T().Fatalf("Expected a status of %d, got %d", e, a)
	}
}

func makeTokenString(SigningAlgorithm string, identity string) string {
	if SigningAlgorithm == "" {
		SigningAlgorithm = "HS256"
	}
	token := jwt.New(jwt.GetSigningMethod(SigningAlgorithm))
	claims := token.Claims.(jwt.MapClaims)
	claims["sub"] = identity // subject
	claims["exp"] = time.Now().Add(time.Hour).Unix()
	claims["orig_iat"] = time.Now().Unix()
	// config := configuration.Get()
	tss, _ := token.SigningString()
	fmt.Printf("Submitted signing string: '%v'\n", tss)
	tokenString, _ := token.SignedString(api.Key)
	return tokenString
}
