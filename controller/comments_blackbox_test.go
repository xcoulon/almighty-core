package controller_test

import (
	"context"
	"fmt"
	"html"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/almighty/almighty-core/account"
	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/comment"
	. "github.com/almighty/almighty-core/controller"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/gormsupport"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/gormtestsupport"
	"github.com/almighty/almighty-core/rendering"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/rest"
	"github.com/almighty/almighty-core/space"
	testsupport "github.com/almighty/almighty-core/test"
	almtoken "github.com/almighty/almighty-core/token"
	"github.com/almighty/almighty-core/workitem"

	"github.com/goadesign/goa"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// a normal test function that will kick off TestSuiteComments
func TestSuiteComments(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &CommentsSuite{DBTestSuite: gormtestsupport.NewDBTestSuite("../config.yaml")})
}

// ========== TestSuiteComments struct that implements SetupSuite, TearDownSuite, SetupTest, TearDownTest ==========
type CommentsSuite struct {
	gormtestsupport.DBTestSuite
	db            *gormapplication.GormDB
	clean         func()
	testIdentity  account.Identity
	testIdentity2 account.Identity
}

func (s *CommentsSuite) SetupTest() {
	s.db = gormapplication.NewGormDB(s.DB)
	s.clean = cleaner.DeleteCreatedEntities(s.DB)
	testIdentity, err := testsupport.CreateTestIdentity(s.DB, "CommentsSuite user", "test provider")
	require.Nil(s.T(), err)
	s.testIdentity = testIdentity
	testIdentity2, err := testsupport.CreateTestIdentity(s.DB, "CommentsSuite user2", "test provider")
	require.Nil(s.T(), err)
	s.testIdentity2 = testIdentity2
}

func (s *CommentsSuite) TearDownTest() {
	s.clean()
}

var (
	markdownMarkup  = rendering.SystemMarkupMarkdown
	plaintextMarkup = rendering.SystemMarkupPlainText
	defaultMarkup   = rendering.SystemMarkupDefault
)

func (s *CommentsSuite) unsecuredController() (*goa.Service, *CommentsController) {
	svc := goa.New("Comments-service-test")
	commentsCtrl := NewCommentsController(svc, s.db, s.Configuration)
	return svc, commentsCtrl
}

func (s *CommentsSuite) securedControllers(identity account.Identity) (*goa.Service, *WorkitemController, *WorkItemCommentsController, *CommentsController) {
	priv, _ := almtoken.ParsePrivateKey([]byte(almtoken.RSAPrivateKey))
	svc := testsupport.ServiceAsUser("Comment-Service", almtoken.NewManagerWithPrivateKey(priv), identity)
	workitemCtrl := NewWorkitemController(svc, s.db, s.Configuration)
	workitemCommentsCtrl := NewWorkItemCommentsController(svc, s.db, s.Configuration)
	commentsCtrl := NewCommentsController(svc, s.db, s.Configuration)
	return svc, workitemCtrl, workitemCommentsCtrl, commentsCtrl
}

// createWorkItem creates a workitem that will be used to perform the comment operations during the tests.
func (s *CommentsSuite) createWorkItem(identity account.Identity) string {
	spaceSelfURL := rest.AbsoluteURL(&goa.RequestData{
		Request: &http.Request{Host: "api.service.domain.org"},
	}, app.SpaceHref(space.SystemSpace.String()))
	witSelfURL := rest.AbsoluteURL(&goa.RequestData{
		Request: &http.Request{Host: "api.service.domain.org"},
	}, app.WorkitemtypeHref(space.SystemSpace.String(), workitem.SystemBug.String()))
	createWorkitemPayload := app.CreateWorkitemPayload{
		Data: &app.WorkItem{
			Type: APIStringTypeWorkItem,
			Attributes: map[string]interface{}{
				workitem.SystemTitle: "work item title",
				workitem.SystemState: workitem.SystemStateNew},
			Relationships: &app.WorkItemRelationships{
				BaseType: &app.RelationBaseType{
					Data: &app.BaseTypeData{
						Type: "workitemtypes",
						ID:   workitem.SystemBug,
					},
					Links: &app.GenericLinks{
						Self: &witSelfURL,
					},
				},
				Space: app.NewSpaceRelation(space.SystemSpace, spaceSelfURL),
			},
		},
	}
	userSvc, workitemCtrl, _, _ := s.securedControllers(identity)
	_, wi := test.CreateWorkitemCreated(s.T(), userSvc.Context, userSvc, workitemCtrl, *createWorkitemPayload.Data.Relationships.Space.Data.ID, &createWorkitemPayload)
	wiID := *wi.Data.ID
	s.T().Log(fmt.Sprintf("Created workitem with id %v", wiID))
	return wiID
}

func (s *CommentsSuite) newCreateWorkItemCommentsPayload(body string, markup *string) *app.CreateWorkItemCommentsPayload {
	return &app.CreateWorkItemCommentsPayload{
		Data: &app.CreateComment{
			Type: "comments",
			Attributes: &app.CreateCommentAttributes{
				Body:   body,
				Markup: markup,
			},
		},
	}
}

func (s *CommentsSuite) newUpdateCommentsPayload(body string, markup *string) *app.UpdateCommentsPayload {
	return &app.UpdateCommentsPayload{
		Data: &app.Comment{
			Type: "comments",
			Attributes: &app.CommentAttributes{
				Body:   &body,
				Markup: markup,
			},
		},
	}
}

// createWorkItemComment creates a workitem comment that will be used to perform the comment operations during the tests.
func (s *CommentsSuite) createWorkItemComment(identity account.Identity, wiID string, body string, markup *string) app.CommentSingle {
	createWorkItemCommentPayload := s.newCreateWorkItemCommentsPayload(body, markup)
	userSvc, _, workitemCommentsCtrl, _ := s.securedControllers(identity)
	_, c := test.CreateWorkItemCommentsOK(s.T(), userSvc.Context, userSvc, workitemCommentsCtrl, space.SystemSpace, wiID, createWorkItemCommentPayload)
	s.T().Log(fmt.Sprintf("Created comment with id %v", *c.Data.ID))
	return *c
}

func assertComment(t *testing.T, resultData *app.Comment, expectedIdentity account.Identity, expectedBody string, expectedMarkup string) {
	require.NotNil(t, resultData)
	assert.NotNil(t, resultData.ID)
	assert.NotNil(t, resultData.Type)
	require.NotNil(t, resultData.Attributes)
	require.NotNil(t, resultData.Attributes.CreatedAt)
	require.NotNil(t, resultData.Attributes.UpdatedAt)
	require.NotNil(t, resultData.Attributes.Body)
	require.NotNil(t, resultData.Attributes.Body)
	assert.Equal(t, expectedBody, *resultData.Attributes.Body)
	require.NotNil(t, resultData.Attributes.Markup)
	assert.Equal(t, expectedMarkup, *resultData.Attributes.Markup)
	assert.Equal(t, rendering.RenderMarkupToHTML(html.EscapeString(expectedBody), expectedMarkup), *resultData.Attributes.BodyRendered)
	require.NotNil(t, resultData.Relationships)
	require.NotNil(t, resultData.Relationships.CreatedBy)
	require.NotNil(t, resultData.Relationships.CreatedBy.Data)
	require.NotNil(t, resultData.Relationships.CreatedBy.Data.ID)
	assert.Equal(t, expectedIdentity.ID, *resultData.Relationships.CreatedBy.Data.ID)
	assert.True(t, strings.Contains(*resultData.Relationships.CreatedBy.Links.Related, resultData.Relationships.CreatedBy.Data.ID.String()), "Link not found")
}

func convertCommentToModel(c app.CommentSingle) comment.Comment {
	return comment.Comment{
		ID: *c.Data.ID,
		Lifecycle: gormsupport.Lifecycle{
			UpdatedAt: *c.Data.Attributes.UpdatedAt,
		},
	}
}

func (s *CommentsSuite) TestShowCommentWithoutAuthOK() {
	// given
	wiID := s.createWorkItem(s.testIdentity)
	c := s.createWorkItemComment(s.testIdentity, wiID, "body", &markdownMarkup)
	// when
	userSvc, commentsCtrl := s.unsecuredController()
	_, result := test.ShowCommentsOK(s.T(), userSvc.Context, userSvc, commentsCtrl, *c.Data.ID, nil, nil)
	// then
	assertComment(s.T(), result.Data, s.testIdentity, "body", rendering.SystemMarkupMarkdown)
}

func (s *CommentsSuite) TestShowCommentWithoutAuthOKUsingExpiredIfModifiedSinceHeader() {
	// given
	wiID := s.createWorkItem(s.testIdentity)
	c := s.createWorkItemComment(s.testIdentity, wiID, "body", &markdownMarkup)
	// when
	ifModifiedSince := app.ToHTTPTime(c.Data.Attributes.UpdatedAt.Add(-1 * time.Hour))
	userSvc, commentsCtrl := s.unsecuredController()
	res, result := test.ShowCommentsOK(s.T(), userSvc.Context, userSvc, commentsCtrl, *c.Data.ID, &ifModifiedSince, nil)
	// then
	assertComment(s.T(), result.Data, s.testIdentity, "body", rendering.SystemMarkupMarkdown)
	assertResponseHeaders(s.T(), res)
}

func (s *CommentsSuite) TestShowCommentWithoutAuthOKUsingExpiredIfNoneMatchHeader() {
	// given
	wiID := s.createWorkItem(s.testIdentity)
	c := s.createWorkItemComment(s.testIdentity, wiID, "body", &markdownMarkup)
	// when
	ifNoneMatch := "foo"
	userSvc, commentsCtrl := s.unsecuredController()
	res, result := test.ShowCommentsOK(s.T(), userSvc.Context, userSvc, commentsCtrl, *c.Data.ID, nil, &ifNoneMatch)
	// then
	assertComment(s.T(), result.Data, s.testIdentity, "body", rendering.SystemMarkupMarkdown)
	assertResponseHeaders(s.T(), res)
}

func (s *CommentsSuite) TestShowCommentWithoutAuthNotModifiedUsingIfModifiedSinceHeader() {
	// given
	wiID := s.createWorkItem(s.testIdentity)
	c := s.createWorkItemComment(s.testIdentity, wiID, "body", &markdownMarkup)
	// when
	ifModifiedSince := app.ToHTTPTime(*c.Data.Attributes.UpdatedAt)
	userSvc, commentsCtrl := s.unsecuredController()
	res := test.ShowCommentsNotModified(s.T(), userSvc.Context, userSvc, commentsCtrl, *c.Data.ID, &ifModifiedSince, nil)
	// then
	assertResponseHeaders(s.T(), res)
}

func (s *CommentsSuite) TestShowCommentWithoutAuthNotModifiedUsingIfNoneMatchHeader() {
	// given
	wiID := s.createWorkItem(s.testIdentity)
	c := s.createWorkItemComment(s.testIdentity, wiID, "body", &markdownMarkup)
	// when
	commentModel := convertCommentToModel(c)
	ifNoneMatch := app.GenerateEntityTag(commentModel)
	userSvc, commentsCtrl := s.unsecuredController()
	res := test.ShowCommentsNotModified(s.T(), userSvc.Context, userSvc, commentsCtrl, *c.Data.ID, nil, &ifNoneMatch)
	// then
	assertResponseHeaders(s.T(), res)
}

func (s *CommentsSuite) TestShowCommentWithoutAuthWithMarkup() {
	// given
	wiID := s.createWorkItem(s.testIdentity)
	c := s.createWorkItemComment(s.testIdentity, wiID, "body", nil)
	// when
	userSvc, commentsCtrl := s.unsecuredController()
	_, result := test.ShowCommentsOK(s.T(), userSvc.Context, userSvc, commentsCtrl, *c.Data.ID, nil, nil)
	// then
	assertComment(s.T(), result.Data, s.testIdentity, "body", rendering.SystemMarkupPlainText)
}

func (s *CommentsSuite) TestShowCommentWithAuth() {
	// given
	wiID := s.createWorkItem(s.testIdentity)
	c := s.createWorkItemComment(s.testIdentity, wiID, "body", &plaintextMarkup)
	// when
	userSvc, _, _, commentsCtrl := s.securedControllers(s.testIdentity)
	_, result := test.ShowCommentsOK(s.T(), userSvc.Context, userSvc, commentsCtrl, *c.Data.ID, nil, nil)
	// then
	assertComment(s.T(), result.Data, s.testIdentity, "body", rendering.SystemMarkupPlainText)
}

func (s *CommentsSuite) TestShowCommentWithEscapedScriptInjection() {
	// given
	wiID := s.createWorkItem(s.testIdentity)
	c := s.createWorkItemComment(s.testIdentity, wiID, "<img src=x onerror=alert('body') />", &plaintextMarkup)
	// when
	userSvc, _, _, commentsCtrl := s.securedControllers(s.testIdentity)
	_, result := test.ShowCommentsOK(s.T(), userSvc.Context, userSvc, commentsCtrl, *c.Data.ID, nil, nil)
	// then
	assertComment(s.T(), result.Data, s.testIdentity, "<img src=x onerror=alert('body') />", rendering.SystemMarkupPlainText)
}

func (s *CommentsSuite) TestUpdateCommentWithoutAuth() {
	// given
	wiID := s.createWorkItem(s.testIdentity)
	c := s.createWorkItemComment(s.testIdentity, wiID, "body", &plaintextMarkup)
	// when
	updateCommentPayload := s.newUpdateCommentsPayload("updated body", &markdownMarkup)
	userSvc, commentsCtrl := s.unsecuredController()
	test.UpdateCommentsUnauthorized(s.T(), userSvc.Context, userSvc, commentsCtrl, *c.Data.ID, updateCommentPayload)
}

func (s *CommentsSuite) TestUpdateCommentWithSameUserWithOtherMarkup() {
	// given
	wiID := s.createWorkItem(s.testIdentity)
	c := s.createWorkItemComment(s.testIdentity, wiID, "body", &plaintextMarkup)
	// when
	updateCommentPayload := s.newUpdateCommentsPayload("updated body", &markdownMarkup)
	userSvc, _, _, commentsCtrl := s.securedControllers(s.testIdentity)
	_, result := test.UpdateCommentsOK(s.T(), userSvc.Context, userSvc, commentsCtrl, *c.Data.ID, updateCommentPayload)
	assertComment(s.T(), result.Data, s.testIdentity, "updated body", rendering.SystemMarkupMarkdown)
}

func (s *CommentsSuite) TestUpdateCommentWithSameUserWithNilMarkup() {
	// given
	wiID := s.createWorkItem(s.testIdentity)
	c := s.createWorkItemComment(s.testIdentity, wiID, "body", &plaintextMarkup)
	// when
	updateCommentPayload := s.newUpdateCommentsPayload("updated body", nil)
	userSvc, _, _, commentsCtrl := s.securedControllers(s.testIdentity)
	_, result := test.UpdateCommentsOK(s.T(), userSvc.Context, userSvc, commentsCtrl, *c.Data.ID, updateCommentPayload)
	assertComment(s.T(), result.Data, s.testIdentity, "updated body", rendering.SystemMarkupDefault)
}

func (s *CommentsSuite) TestUpdateCommentWithOtherUser() {
	// given
	wiID := s.createWorkItem(s.testIdentity)
	c := s.createWorkItemComment(s.testIdentity, wiID, "body", &plaintextMarkup)
	// when
	updatedCommentBody := "An updated comment"
	updateCommentPayload := &app.UpdateCommentsPayload{
		Data: &app.Comment{
			Type: "comments",
			Attributes: &app.CommentAttributes{
				Body: &updatedCommentBody,
			},
		},
	}
	// when/then
	userSvc, _, _, commentsCtrl := s.securedControllers(s.testIdentity2)
	test.UpdateCommentsForbidden(s.T(), userSvc.Context, userSvc, commentsCtrl, *c.Data.ID, updateCommentPayload)
}

func (s *CommentsSuite) TestDeleteCommentWithSameAuthenticatedUser() {
	// given
	wiID := s.createWorkItem(s.testIdentity)
	c := s.createWorkItemComment(s.testIdentity, wiID, "body", &plaintextMarkup)
	userSvc, _, _, commentsCtrl := s.securedControllers(s.testIdentity)
	// when/then
	test.DeleteCommentsOK(s.T(), userSvc.Context, userSvc, commentsCtrl, *c.Data.ID)
}

func (s *CommentsSuite) TestDeleteCommentWithoutAuth() {
	// given
	wiID := s.createWorkItem(s.testIdentity)
	c := s.createWorkItemComment(s.testIdentity, wiID, "body", &plaintextMarkup)
	userSvc, commentsCtrl := s.unsecuredController()
	// when/then
	test.DeleteCommentsUnauthorized(s.T(), userSvc.Context, userSvc, commentsCtrl, *c.Data.ID)
}

func (s *CommentsSuite) TestDeleteCommentWithOtherAuthenticatedUser() {
	// given
	wiID := s.createWorkItem(s.testIdentity)
	c := s.createWorkItemComment(s.testIdentity, wiID, "body", &plaintextMarkup)
	userSvc, _, _, commentsCtrl := s.securedControllers(s.testIdentity2)
	// when/then
	test.DeleteCommentsForbidden(s.T(), userSvc.Context, userSvc, commentsCtrl, *c.Data.ID)
}

func (s *CommentsSuite) TestAddCommentOnWorkItem() {
	// given a work item with a comment
	wiID := s.createWorkItem(s.testIdentity)
	workitemRepo := workitem.NewWorkItemRepository(s.DB)
	wi, err := workitemRepo.LoadByID(context.Background(), wiID)
	require.Nil(s.T(), err)
	commentedAt, ok := wi.Fields[workitem.SystemCommentedAt]
	require.True(s.T(), ok)
	require.Nil(s.T(), commentedAt)
	// when
	c := s.createWorkItemComment(s.testIdentity, wiID, "body", &plaintextMarkup)
	// then load the work item again and verify the timestamps
	wi, err = workitemRepo.LoadByID(context.Background(), wiID)
	require.Nil(s.T(), err)
	commentedAt, ok = wi.Fields[workitem.SystemCommentedAt]
	require.True(s.T(), ok)
	require.NotNil(s.T(), commentedAt)
	assert.Equal(s.T(), c.Data.Attributes.CreatedAt.UTC(), commentedAt.(*time.Time).UTC())
}

func (s *CommentsSuite) TestAddAndRemoveCommentOnWorkItem() {
	// given a work item with a (soft) delete comment
	wiID := s.createWorkItem(s.testIdentity)
	workitemRepo := workitem.NewWorkItemRepository(s.DB)
	wi, err := workitemRepo.LoadByID(context.Background(), wiID)
	require.Nil(s.T(), err)
	commentedAt, ok := wi.Fields[workitem.SystemCommentedAt]
	require.True(s.T(), ok)
	require.Nil(s.T(), commentedAt)
	// when
	c := s.createWorkItemComment(s.testIdentity, wiID, "body", &plaintextMarkup)
	userSvc, _, _, commentsCtrl := s.securedControllers(s.testIdentity)
	time.Sleep(1 * time.Second)
	test.DeleteCommentsOK(s.T(), userSvc.Context, userSvc, commentsCtrl, *c.Data.ID)
	// then load the work item again and verify the timestamps
	wi, err = workitemRepo.LoadByID(context.Background(), wiID)
	require.Nil(s.T(), err)
	commentedAt, ok = wi.Fields[workitem.SystemCommentedAt]
	require.True(s.T(), ok)
	require.NotNil(s.T(), commentedAt)
	assert.True(s.T(), commentedAt.(*time.Time).After(*c.Data.Attributes.CreatedAt))
}
