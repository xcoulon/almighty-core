package handler_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo"

	"github.com/fabric8-services/fabric8-wit/api/model"
	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/google/jsonapi"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type SpacesResourceBlackBoxTestSuite struct {
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

type SpacesResourceTestSuite struct {
	gormtestsupport.GinkgoTestSuite
	ctx context.Context
}

var _ = Describe("Spaces", func() {

	var s SpacesResourceTestSuite

	BeforeEach(func() {
		s = SpacesResourceTestSuite{GinkgoTestSuite: gormtestsupport.NewGinkgoTestSuite("../../config.yaml")}
		s.Clean = cleaner.DeleteCreatedEntities(s.DB)
		var err error
		require.Nil(GinkgoT(), err)
	})

	AfterEach(func() {
		s.Clean()
		s.TearDown()
	})

	Describe("Test Spaces", func() {
		Context("Create Space", func() {

			Specify("Create Space OK", func() {
				// given
				name := "test-space-" + uuid.NewV4().String()
				description := "A description"
				payloadSpace := model.Space{
					Name:        name,
					Description: description,
				}
				// when
				rr := CreateSpace(&s.GinkgoTestSuite, &payloadSpace, s.TestUser1())
				// then
				assert.Equal(GinkgoT(), http.StatusCreated, rr.Code)
				GinkgoT().Logf("Response body:\n%s", rr.Body.String())
				responseItem := model.Space{}
				err := jsonapi.UnmarshalPayload(rr.Body, &responseItem)
				require.Nil(GinkgoT(), err)
				assert.NotNil(GinkgoT(), responseItem.ID)
				assert.Equal(GinkgoT(), "A description", responseItem.Description)
			})

			Specify("Create Space KO - missing credentials", func() {
				// given
				name := "test-space-" + uuid.NewV4().String()
				description := "A description"
				payloadSpace := model.Space{
					Name:        name,
					Description: description,
				}
				rr := CreateSpace(&s.GinkgoTestSuite, &payloadSpace, nil)
				// then
				assert.Equal(GinkgoT(), http.StatusUnauthorized, rr.Code)
			})

		})

	})
})

func CreateSpace(suite *gormtestsupport.GinkgoTestSuite, payloadSpace *model.Space, testuser *gormtestsupport.TestUser) *httptest.ResponseRecorder {
	payload := bytes.NewBuffer(make([]byte, 0))
	err := jsonapi.MarshalPayload(payload, payloadSpace)
	require.Nil(GinkgoT(), err)
	r, _ := http.NewRequest(http.MethodPost, "/api/spaces/", payload)
	require.Nil(GinkgoT(), err)
	if testuser != nil {
		r.Header.Set("Authorization", "Bearer "+testuser.AccessToken)
	}
	return Execute(*suite, r)
}
