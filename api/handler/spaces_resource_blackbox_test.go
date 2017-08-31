package handler_test

import (
	"bytes"
	"context"
	"net/http"

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
	clean func()
	ctx   context.Context
}

var accessToken *string
var _ = Describe("Spaces", func() {

	s := SpacesResourceTestSuite{GinkgoTestSuite: gormtestsupport.NewGinkgoTestSuite("../../config.yaml")}
	BeforeEach(func() {
		s.Setup()
		s.clean = cleaner.DeleteCreatedEntities(s.DB)
		var err error
		_, _, accessToken, err = s.GenerateTestUserIdentityAndToken(s.Configuration.GetKeycloakTestUserName(), s.Configuration.GetKeycloakTestUserSecret())
		require.Nil(GinkgoT(), err)
		GinkgoT().Logf("Generated access token: %s\n", *accessToken)
	})

	AfterEach(func() {
		// s.clean()
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
				payload := bytes.NewBuffer(make([]byte, 0))
				err := jsonapi.MarshalPayload(payload, &payloadSpace)
				require.Nil(GinkgoT(), err)
				r, _ := http.NewRequest(http.MethodPost, "/api/spaces/", payload)
				require.Nil(GinkgoT(), err)
				r.Header.Set("Authorization", "Bearer "+*accessToken)
				// when
				rr := Execute(s.GinkgoTestSuite, r)
				// then
				assert.Equal(GinkgoT(), http.StatusCreated, rr.Code)
				responseItem := model.Space{}
				GinkgoT().Logf("Response body:\n%s", rr.Body.String())
				err = jsonapi.UnmarshalPayload(rr.Body, &responseItem)
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
				payload := bytes.NewBuffer(make([]byte, 0))
				err := jsonapi.MarshalPayload(payload, &payloadSpace)
				require.Nil(GinkgoT(), err)
				r, _ := http.NewRequest(http.MethodPost, "/api/spaces/", payload)
				require.Nil(GinkgoT(), err)
				// when
				rr := Execute(s.GinkgoTestSuite, r)
				// then
				assert.Equal(GinkgoT(), http.StatusUnauthorized, rr.Code)
			})

		})

	})
})
