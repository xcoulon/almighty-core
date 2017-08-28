package handler_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/api/model"
	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	"github.com/fabric8-services/fabric8-wit/rendering"
	"github.com/fabric8-services/fabric8-wit/space"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/google/jsonapi"
	. "github.com/onsi/ginkgo"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type WorkItemsResourceTestSuite struct {
	gormtestsupport.GinkgoTestSuite
	clean func()
	ctx   context.Context
	space space.Space
}

var _ = Describe("WorkItems", func() {

	s := WorkItemsResourceTestSuite{GinkgoTestSuite: gormtestsupport.NewGinkgoTestSuite("../../config.yaml")}

	BeforeSuite(func() {
		s.Setup()
		// also, create a testing space for all operations
		spaceOwnerIdentity := createOneRandomUserIdentity(context.Background(), s.DB)
		spaceRepo := space.NewRepository(s.DB)
		testSpace, err := spaceRepo.Create(context.Background(), &space.Space{
			Name:        "test-" + uuid.NewV4().String(),
			Description: "Test space",
			OwnerId:     spaceOwnerIdentity.ID,
		})
		require.Nil(GinkgoT(), err)
		s.space = *testSpace
	})

	AfterSuite(func() {
		s.TearDown()
	})

	BeforeEach(func() {
		s.clean = cleaner.DeleteCreatedEntities(s.DB)
	})

	AfterEach(func() {
		// s.clean()
	})

	Describe("Test WorkItems", func() {
		Context("Create WorkItem", func() {

			Specify("Create WorkItem OK", func() {
				// given
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
				r, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/spaces/%[1]s/workitems", s.space.ID.String()), payload)
				r.Header.Set("Authorization", "Bearer "+makeTokenString("HS256", testIdentity.ID.String(), nil))
				// when
				rr := Execute(s.GinkgoTestSuite, r)
				// then
				assert.Equal(GinkgoT(), http.StatusCreated, rr.Code)
				responseItem := model.WorkItem{}
				GinkgoT().Logf("Response body:\n%s", rr.Body.String())
				err = jsonapi.UnmarshalPayload(rr.Body, &responseItem)
				require.Nil(GinkgoT(), err)
				assert.NotNil(GinkgoT(), responseItem.ID)
				assert.Equal(GinkgoT(), "A description", *responseItem.Description)
			})

			Specify("Create WorkItem KO - missing JWT", func() {
				// given
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
				r, err := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/spaces/%[1]s/workitems", s.space.ID.String()), payload)
				// when
				rr := Execute(s.GinkgoTestSuite, r)
				// then
				assert.Equal(GinkgoT(), http.StatusUnauthorized, rr.Code)
			})

			Specify("Create WorkItem KO - invalid credentials", func() {
				// given
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
				r, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/spaces/%[1]s/workitems", s.space.ID.String()), payload)
				// generate/sign an auth token
				r.Header.Set("Authorization", "Bearer "+makeTokenString("HS256", "foo", nil))
				// when
				rr := Execute(s.GinkgoTestSuite, r)
				// then
				assert.Equal(GinkgoT(), http.StatusForbidden, rr.Code)
			})
		})

		Context("Update WorkItem", func() {

			var workitemCreatorIdentity *account.Identity
			var spaceEditorIdentity *account.Identity
			var createdWorkItem workitem.WorkItem

			BeforeEach(func() {
				// given
				workitemCreatorIdentity = createOneRandomUserIdentity(context.Background(), s.DB)
				spaceEditorIdentity = createOneRandomUserIdentity(context.Background(), s.DB)
				// create a bunch a work items
				workitemRepo := workitem.NewWorkItemRepository(s.DB)
				wiFields := map[string]interface{}{
					workitem.SystemTitle: "test work item",
					workitem.SystemDescription: rendering.MarkupContent{
						Content: "test work item description",
					},
					workitem.SystemState: workitem.SystemStateNew,
				}
				createdWI, err := workitemRepo.Create(context.Background(), s.space.ID, workitem.SystemBug, wiFields, workitemCreatorIdentity.ID)
				require.Nil(GinkgoT(), err)
				createdWorkItem = *createdWI
				GinkgoT().Logf("Created work item with id='%s' in space '%s'", createdWorkItem.ID.String(), createdWorkItem.SpaceID.String())
			})

			Specify("Update WorkItem OK - work item creator", func() {
				// given
				payloadWI := model.ConvertWorkItemToModel(createdWorkItem)
				updatedTitle := "Updated title"
				payloadWI.Title = &updatedTitle
				payload := bytes.NewBuffer(make([]byte, 0))
				err := jsonapi.MarshalPayload(payload, payloadWI)
				require.Nil(GinkgoT(), err)
				r, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/api/workitems/%[1]s", createdWorkItem.ID.String()), payload)
				r.Header.Set("Authorization", "Bearer "+makeTokenString("HS256", workitemCreatorIdentity.ID.String(), nil))
				// when
				rr := Execute(s.GinkgoTestSuite, r)
				// then
				assert.Equal(GinkgoT(), http.StatusOK, rr.Code)
				responseItem := model.WorkItem{}
				GinkgoT().Logf("Response body:\n%s", rr.Body.String())
				err = jsonapi.UnmarshalPayload(rr.Body, &responseItem)
				require.Nil(GinkgoT(), err)
				assert.NotNil(GinkgoT(), responseItem.ID)
				assert.Equal(GinkgoT(), "Updated title", *responseItem.Title)
			})

			Specify("Update WorkItem OK - space editor", func() {
				// given
				payloadWI := model.ConvertWorkItemToModel(createdWorkItem)
				updatedTitle := "Updated title"
				payloadWI.Title = &updatedTitle
				payload := bytes.NewBuffer(make([]byte, 0))
				err := jsonapi.MarshalPayload(payload, payloadWI)
				require.Nil(GinkgoT(), err)
				r, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/api/workitems/%[1]s", createdWorkItem.ID.String()), payload)
				r.Header.Set("Authorization", "Bearer "+makeTokenString("HS256", spaceEditorIdentity.ID.String(), nil))
				// when
				rr := Execute(s.GinkgoTestSuite, r)
				// then
				assert.Equal(GinkgoT(), http.StatusOK, rr.Code)
				responseItem := model.WorkItem{}
				GinkgoT().Logf("Response body:\n%s", rr.Body.String())
				err = jsonapi.UnmarshalPayload(rr.Body, &responseItem)
				require.Nil(GinkgoT(), err)
				assert.NotNil(GinkgoT(), responseItem.ID)
				assert.Equal(GinkgoT(), "Updated title", *responseItem.Title)
			})

			Specify("Update WorkItem KO - invalid credentials", func() {
				// given
				payloadWI := model.ConvertWorkItemToModel(createdWorkItem)
				updatedTitle := "Updated title"
				payloadWI.Title = &updatedTitle
				payload := bytes.NewBuffer(make([]byte, 0))
				err := jsonapi.MarshalPayload(payload, payloadWI)
				require.Nil(GinkgoT(), err)
				r, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/api/workitems/%[1]s", createdWorkItem.ID.String()), payload)
				// generate an invalid auth token
				r.Header.Set("Authorization", "Bearer "+makeTokenString("HS256", "foo", nil))
				// when
				rr := Execute(s.GinkgoTestSuite, r)
				// then
				assert.Equal(GinkgoT(), http.StatusForbidden, rr.Code)
			})
		})

		Context("List WorkItems", func() {

			var testIdentity *account.Identity
			createdWorkItems := make([]workitem.WorkItem, 10)

			BeforeEach(func() {
				GinkgoT().Log("creating a set of work items to test the updates...")
				// given
				testIdentity = createOneRandomUserIdentity(context.Background(), s.DB)
				// create a bunch a work items
				workitemRepo := workitem.NewWorkItemRepository(s.DB)
				for i := 0; i < 10; i++ {
					wiFields := map[string]interface{}{
						workitem.SystemTitle: fmt.Sprintf("test work item #%d", i),
						workitem.SystemDescription: rendering.MarkupContent{
							Content: fmt.Sprintf("test work item #%d description", i),
						},
						workitem.SystemState: workitem.SystemStateNew,
					}
					createdWI, err := workitemRepo.Create(context.Background(), s.space.ID, workitem.SystemBug, wiFields, s.space.OwnerId)
					require.Nil(GinkgoT(), err)
					createdWorkItems[i] = *createdWI
				}
			})

			Specify("List WorkItems OK", func() {
				// given
				r, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/spaces/%[1]s/workitems", s.space.ID.String()), nil)
				// when
				rr := Execute(s.GinkgoTestSuite, r)
				// then
				assert.Equal(GinkgoT(), http.StatusOK, rr.Code)
			})
		})

		Context("Show WorkItems", func() {

			var testIdentity *account.Identity
			var createdWorkItem workitem.WorkItem

			BeforeEach(func() {
				// given
				testIdentity = createOneRandomUserIdentity(context.Background(), s.DB)
				// create a bunch a work items
				workitemRepo := workitem.NewWorkItemRepository(s.DB)
				wiFields := map[string]interface{}{
					workitem.SystemTitle: "test work item",
					workitem.SystemDescription: rendering.MarkupContent{
						Content: "test work item description",
					},
					workitem.SystemState: workitem.SystemStateNew,
				}
				createdWI, err := workitemRepo.Create(context.Background(), s.space.ID, workitem.SystemBug, wiFields, s.space.OwnerId)
				require.Nil(GinkgoT(), err)
				createdWorkItem = *createdWI
			})

			Specify("Show WorkItem OK", func() {
				// given
				r, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/workitems/%s", createdWorkItem.ID.String()), nil)
				// when
				rr := Execute(s.GinkgoTestSuite, r)
				// then
				assert.Equal(GinkgoT(), http.StatusOK, rr.Code)
			})
		})

	})

})
