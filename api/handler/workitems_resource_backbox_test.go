package handler_test

import (
	"bytes"
	"context"
	"net/http"

	"github.com/fabric8-services/fabric8-wit/api/model"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/google/jsonapi"
	. "github.com/onsi/ginkgo"
	"github.com/stretchr/testify/require"
)

type WorkItemsResourceTestSuite struct {
	gormtestsupport.GinkgoTestSuite
	clean func()
	repo  workitem.WorkItemTypeRepository
	ctx   context.Context
}

var _ = Describe("WorkItems", func() {

	s := WorkItemsResourceTestSuite{GinkgoTestSuite: gormtestsupport.NewGinkgoTestSuite("../../config.yaml")}

	BeforeEach(func() {
		s.Setup()
	})

	AfterEach(func() {
		s.TearDown()
	})

	Describe("Test WorkItems", func() {
		It("List WorkItems OK", func() {
			r, err := http.NewRequest(http.MethodGet, "/api/spaces/2e0698d8-753e-4cef-bb7c-f027634824a2/workitems", nil)
			require.Nil(GinkgoT(), err)
			verify(s.GinkgoTestSuite, r, http.StatusOK)
		})

		It("Show WorkItem", func() {
			r, err := http.NewRequest(http.MethodGet, "/api/workitems/c870914b-7942-4b87-8271-3afda49004e0", nil)
			require.Nil(GinkgoT(), err)
			verify(s.GinkgoTestSuite, r, http.StatusOK)
		})

		Context("WorkItem creation", func() {

			It("Create WorkItem", func() {
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
				require.Nil(GinkgoT(), err)
				r, err := http.NewRequest(http.MethodPost, "/api/spaces/2e0698d8-753e-4cef-bb7c-f027634824a2/workitems", payload)
				require.Nil(GinkgoT(), err) // generate/sign an auth token
				r.Header.Set("Authorization", "Bearer "+makeTokenString("HS256", testIdentity.ID.String()))
				verify(s.GinkgoTestSuite, r, http.StatusCreated)
			})

			It("Create WorkItem KO - missing JWT", func() {
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
				require.Nil(GinkgoT(), err)
				r, err := http.NewRequest(http.MethodPost, "/api/spaces/2e0698d8-753e-4cef-bb7c-f027634824a2/workitems", payload)
				require.Nil(GinkgoT(), err)
				verify(s.GinkgoTestSuite, r, http.StatusUnauthorized)
			})

			It("Create WorkItem KO - invalid credentials", func() {
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
				require.Nil(GinkgoT(), err)
				r, err := http.NewRequest(http.MethodPost, "/api/spaces/2e0698d8-753e-4cef-bb7c-f027634824a2/workitems", payload)
				require.Nil(GinkgoT(), err)
				// generate/sign an auth token
				r.Header.Set("Authorization", "Bearer "+makeTokenString("HS256", "foo"))
				verify(s.GinkgoTestSuite, r, http.StatusForbidden)
			})
		})
	})

})
