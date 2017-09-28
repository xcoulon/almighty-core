package handler_test

import (
	"bytes"
	"context"
	"fmt"
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

type SpacesResourceTestSuite struct {
	gormtestsupport.GinkgoDBTestSuite
	ctx context.Context
}

var _ = Describe("Spaces", func() {

	var s SpacesResourceTestSuite

	BeforeEach(func() {
		s = SpacesResourceTestSuite{GinkgoDBTestSuite: gormtestsupport.NewGinkgoDBTestSuite("../../config.yaml")}
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
					Name:        &name,
					Description: &description,
				}
				// when
				rr := CreateSpace(&s.GinkgoDBTestSuite, &payloadSpace, s.TestUser1())
				// then
				assert.Equal(GinkgoT(), http.StatusCreated, rr.Code)
				GinkgoT().Logf("Response body:\n%s", rr.Body.String())
				responseItem := model.Space{}
				err := jsonapi.UnmarshalPayload(rr.Body, &responseItem)
				require.Nil(GinkgoT(), err)
				assert.NotNil(GinkgoT(), responseItem.ID)
				assert.Equal(GinkgoT(), "A description", *responseItem.Description)
			})

			Specify("Create Space KO - missing credentials", func() {
				// given
				name := "test-space-" + uuid.NewV4().String()
				description := "A description"
				payloadSpace := model.Space{
					Name:        &name,
					Description: &description,
				}
				rr := CreateSpace(&s.GinkgoDBTestSuite, &payloadSpace, nil)
				// then
				assert.Equal(GinkgoT(), http.StatusUnauthorized, rr.Code)
			})

		})

		Context("Show Space", func() {

			var spaceID uuid.UUID

			BeforeEach(func() {
				s.Setup()
				// also, create a testing space for all operations
				name := "test-" + uuid.NewV4().String()
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
				spaceID = *space.ID
				require.Nil(GinkgoT(), err)
			})

			Specify("Show Space OK", func() {
				// given
				r, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/spaces/%s", spaceID.String()), nil)
				// when
				rr := Execute(s.GinkgoDBTestSuite, r)
				// then
				assert.Equal(GinkgoT(), http.StatusOK, rr.Code)
			})

		})

	})
})

func CreateSpace(suite *gormtestsupport.GinkgoDBTestSuite, payloadSpace *model.Space, testuser *gormtestsupport.TestUser) *httptest.ResponseRecorder {
	payload := bytes.NewBuffer(make([]byte, 0))
	err := jsonapi.MarshalPayload(context.Background(), payload, payloadSpace)
	require.Nil(GinkgoT(), err)
	r, _ := http.NewRequest(http.MethodPost, "/api/spaces", payload)
	require.Nil(GinkgoT(), err)
	if testuser != nil {
		r.Header.Set("Authorization", "Bearer "+testuser.AccessToken)
	}
	return Execute(*suite, r)
}
