package handler_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"

	. "github.com/onsi/ginkgo"

	"github.com/fabric8-services/fabric8-wit/api/model"
	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/google/jsonapi"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type NamedspacesResourceBlackBoxTestSuite struct {
	gormtestsupport.DBTestSuite
	clean func()
	ctx   context.Context
}

// func TestSpacesResource(t *testing.T) {
// 	suite.Run(t, &SpacesResourceBlackBoxTestSuite{DBTestSuite: gormtestsupport.NewDBTestSuite("../../config.yaml")})
// }

// func (s *SpacesResourceBlackBoxTestSuite) TestShowSpaceOK() {
// 	r, err := http.NewRequest(http.MethodGet, "/api/spaces/2e0698d8-753e-4cef-bb7c-f027634824a2", nil)
// 	if err != nil {
// 		s.T().Fatal(err)
// 	}
// 	// r.Header.Set(headerAccept, jsonapi.MediaType)

// 	rr := httptest.NewRecorder()
// 	httpEngine := api.NewGinEngine(gormapplication.NewGormDB(s.DB), s.Configuration)
// 	httpEngine.ServeHTTP(rr, r)

// 	if e, a := http.StatusOK, rr.Code; e != a {
// 		s.T().Fatalf("Expected a status of %d, got %d", e, a)
// 	}
// }

type NamedspacesResourceTestSuite struct {
	gormtestsupport.GinkgoDBTestSuite
	ctx context.Context
}

var _ = Describe("Namedspaces", func() {

	var s NamedspacesResourceTestSuite

	BeforeEach(func() {
		s = NamedspacesResourceTestSuite{GinkgoDBTestSuite: gormtestsupport.NewGinkgoDBTestSuite("../../config.yaml")}
		s.Clean = cleaner.DeleteCreatedEntities(s.DB)
		var err error
		require.Nil(GinkgoT(), err)
	})

	AfterEach(func() {
		s.Clean()
		s.TearDown()
	})

	Describe("Test Namedspaces", func() {

		Context("Show Namedspace", func() {

			var spaceName string

			BeforeEach(func() {
				s.Setup()
				// also, create a testing space for all operations
				name := fmt.Sprintf("test-%s", uuid.NewV4().String())
				description := "Test space"
				rr := CreateSpace(&s.GinkgoDBTestSuite, &model.Space{
					Name:        &name,
					Description: &description,
				}, s.TestUser1())
				require.Equal(GinkgoT(), http.StatusCreated, rr.Code)
				responseItem := model.Space{}
				err := jsonapi.UnmarshalPayload(rr.Body, &responseItem)
				require.Nil(GinkgoT(), err)
				space := responseItem
				spaceName = *space.Name
			})

			Specify("Show Namedspace OK", func() {
				// given
				r, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost:8080/api/namedspaces/%s/%s", s.TestUser1().Identity.Username, spaceName), nil)
				// when
				rr := Execute(s.GinkgoDBTestSuite, r)
				// then
				assert.Equal(GinkgoT(), http.StatusOK, rr.Code)
			})

			Specify("Show Namedspace Not Found", func() {
				// given
				r, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost:8080/api/namedspaces/%s/%s", "foo", spaceName), nil)
				// when
				rr := Execute(s.GinkgoDBTestSuite, r)
				// then
				assert.Equal(GinkgoT(), http.StatusNotFound, rr.Code)
			})

		})
		Context("List Namedspaces", func() {

			BeforeEach(func() {
				s.Setup()
				for i := 0; i < 3; i++ {
					// also, create a testing space for all operations
					name := fmt.Sprintf("test-%d-%s", i, uuid.NewV4().String())
					description := fmt.Sprintf("Test space %d", i)
					rr := CreateSpace(&s.GinkgoDBTestSuite, &model.Space{
						Name:        &name,
						Description: &description,
					}, s.TestUser1())
					require.Equal(GinkgoT(), http.StatusCreated, rr.Code)
					responseItem := model.Space{}
					err := jsonapi.UnmarshalPayload(rr.Body, &responseItem)
					require.Nil(GinkgoT(), err)
				}
			})

			Specify("List Namedspaces with default paging OK", func() {
				// given
				r, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost:8080/api/namedspaces/%s", s.TestUser1().Identity.Username), nil)
				// when
				rr := Execute(s.GinkgoDBTestSuite, r)
				// then
				GinkgoT().Logf("Response body:\n%s", rr.Body.String())
				require.Equal(GinkgoT(), http.StatusOK, rr.Code)
				// attempt to unmarshall the response body
				spaces, err := jsonapi.UnmarshalManyPayload(rr.Body, reflect.TypeOf(new(model.Space)))
				require.Nil(GinkgoT(), err)
				require.Nil(GinkgoT(), err)
				assert.Len(GinkgoT(), spaces, 20)

			})

			Specify("List Namedspaces with custom paging OK", func() {
				// given
				r, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost:8080/api/namedspaces/%s?page[offset]=10&page[limit]=1", s.TestUser1().Identity.Username), nil)
				// when
				rr := Execute(s.GinkgoDBTestSuite, r)
				// then
				// GinkgoT().Logf("Response body:\n%s", rr.Body.String())
				require.Equal(GinkgoT(), http.StatusOK, rr.Code)
				genericBody := map[string]interface{}{}
				err := json.Unmarshal(rr.Body.Bytes(), &genericBody)
				require.Nil(GinkgoT(), err)
				indentedOutput, err := json.MarshalIndent(genericBody, "  ", "  ")
				require.Nil(GinkgoT(), err)
				GinkgoT().Logf("Response body:\n%s", string(indentedOutput))
				// attempt to unmarshall the response body
				spaces, err := jsonapi.UnmarshalManyPayload(rr.Body, reflect.TypeOf(new(model.Space)))
				require.Nil(GinkgoT(), err)
				assert.Len(GinkgoT(), spaces, 1)
			})

			Specify("List Namedspaces Not Found", func() {
				// given
				r, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost:8080/api/namedspaces/%s", "foo"), nil)
				// when
				rr := Execute(s.GinkgoDBTestSuite, r)
				// then
				assert.Equal(GinkgoT(), http.StatusNotFound, rr.Code)
			})

		})

	})
})
