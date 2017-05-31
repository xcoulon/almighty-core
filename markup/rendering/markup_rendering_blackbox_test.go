package rendering_test

import (
	"context"
	"strings"
	"testing"

	"fmt"

	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/gormtestsupport"
	"github.com/almighty/almighty-core/markup"
	"github.com/almighty/almighty-core/markup/rendering"
	"github.com/almighty/almighty-core/migration"
	"github.com/almighty/almighty-core/resource"
	"github.com/almighty/almighty-core/space"
	testsupport "github.com/almighty/almighty-core/test"
	"github.com/almighty/almighty-core/workitem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// The WorkItemTypeTestSuite has state the is relevant to all tests.
// It implements these interfaces from the suite package: SetupAllSuite, SetupTestSuite, TearDownAllSuite, TearDownTestSuite
type MarkupRenderingSuite struct {
	gormtestsupport.DBTestSuite
	clean          func()
	markupRenderer rendering.MarkupRenderer
	workitemRepo   workitem.WorkItemRepository
	baseAPI        string
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestMarkupRenderingSuite(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, &MarkupRenderingSuite{
		DBTestSuite: gormtestsupport.NewDBTestSuite(""),
	})
}

// The SetupSuite method will run before the tests in the suite are run.
// It sets up a database connection for all the tests in this suite without polluting global space.
func (s *MarkupRenderingSuite) SetupSuite() {
	s.DBTestSuite.SetupSuite()
	ctx := migration.NewMigrationContext(context.Background())
	s.DBTestSuite.PopulateDBTestSuite(ctx)
}

// The SetupTest method will be run before every test in the suite.
func (s *MarkupRenderingSuite) SetupTest() {
	s.clean = cleaner.DeleteCreatedEntities(s.DB)
	s.workitemRepo = workitem.NewWorkItemRepository(s.DB)
	s.baseAPI = "http://foobar.com"
	s.markupRenderer = rendering.NewMarkupRenderer(s.workitemRepo, s.baseAPI)
}

func (s *MarkupRenderingSuite) TearDownTest() {
	s.clean()
}

func (s *MarkupRenderingSuite) TestRenderMarkdownContent() {
	// given
	content := "Hello, `World`!"
	// when
	result, err := s.markupRenderer.RenderMarkupToHTML(context.Background(), space.SystemSpace, content, markup.SystemMarkupMarkdown)
	// then
	require.Nil(s.T(), err)
	s.T().Log(result)
	require.NotNil(s.T(), result)
	assert.Equal(s.T(), "<p>Hello, <code>World</code>!</p>\n", *result)
}

func (s *MarkupRenderingSuite) TestRenderMarkdownContentWithFence() {
	// given
	content := "``` go\nfunc getTrue() bool {return true}\n```"
	// when
	result, err := s.markupRenderer.RenderMarkupToHTML(context.Background(), space.SystemSpace, content, markup.SystemMarkupMarkdown)
	// then
	require.Nil(s.T(), err)
	s.T().Log(result)
	require.NotNil(s.T(), result)
	assert.True(s.T(), strings.Contains(*result, "<code class=\"prettyprint language-go\">"))
}

func (s *MarkupRenderingSuite) TestRenderMarkdownContentWithFenceHighlighter() {
	// given
	content := "``` go\nfunc getTrue() bool {return true}\n```"
	// when
	result, err := s.markupRenderer.RenderMarkupToHTML(context.Background(), space.SystemSpace, content, markup.SystemMarkupMarkdown)
	// then
	require.Nil(s.T(), err)
	s.T().Log(result)
	require.NotNil(s.T(), result)
	assert.True(s.T(), strings.Contains(*result, "<code class=\"prettyprint language-go\">"))
	assert.True(s.T(), strings.Contains(*result, "<span class=\"kwd\">func</span>"))
}

func (s *MarkupRenderingSuite) TestInsertLinkToExistingIssue() {
	// given
	testIdentity, err := testsupport.CreateTestIdentity(s.DB, "WorkItemSuite setup user", "test provider")
	require.Nil(s.T(), err)
	wi1, err := s.workitemRepo.Create(context.Background(), space.SystemSpace, workitem.SystemBug, map[string]interface{}{
		workitem.SystemTitle: "Test item 1",
		workitem.SystemState: workitem.SystemStateNew,
	}, testIdentity.ID)
	require.Nil(s.T(), err)
	s.T().Log("Created a work item with number ", wi1.Number)
	wi2, err := s.workitemRepo.Create(context.Background(), space.SystemSpace, workitem.SystemBug, map[string]interface{}{
		workitem.SystemTitle: "Test item 2",
		workitem.SystemState: workitem.SystemStateNew,
	}, testIdentity.ID)
	require.Nil(s.T(), err)
	s.T().Log("Created a work item with number ", wi2.Number)
	// when
	content := fmt.Sprintf("Linking to issue #%d and #%d in the content", wi1.Number, wi2.Number)
	result, err := s.markupRenderer.RenderMarkupToHTML(context.Background(), space.SystemSpace, content, markup.SystemMarkupMarkdown)
	// then
	require.Nil(s.T(), err)
	s.T().Log(result)
	require.NotNil(s.T(), result)
	htmlLinkTemplate := "<a href=\"%[1]s/api/spaces/%[2]v/workitems/%[3]v\" title=\"%[4]s\">#%[5]d</a>"
	htmlLinkToWorkitem1 := fmt.Sprintf(htmlLinkTemplate, s.baseAPI, space.SystemSpace, wi1.ID, wi1.Fields[workitem.SystemTitle], wi1.Number)
	htmlLinkToWorkitem2 := fmt.Sprintf(htmlLinkTemplate, s.baseAPI, space.SystemSpace, wi2.ID, wi2.Fields[workitem.SystemTitle], wi2.Number)
	expectedContent := fmt.Sprintf("<p>Linking to issue %[1]s and %[2]s in the content</p>\n", htmlLinkToWorkitem1, htmlLinkToWorkitem2)
	assert.Equal(s.T(), expectedContent, *result)
}

func (s *MarkupRenderingSuite) TestDoNotInsertLinkToNonExistingIssue() {
	// when
	content := "Linking to non existing issue #9999 in the content"
	result, err := s.markupRenderer.RenderMarkupToHTML(context.Background(), space.SystemSpace, content, markup.SystemMarkupMarkdown)
	// then
	require.Nil(s.T(), err)
	s.T().Log(result)
	require.NotNil(s.T(), result)
	expectedContent := "<p>Linking to non existing issue #9999 in the content</p>\n"
	assert.Equal(s.T(), expectedContent, *result)
}

func (s *MarkupRenderingSuite) TestDoNotInsertLinkToEmptyValue() {
	// when
	content := "Linking to non existing issue # in the content"
	result, err := s.markupRenderer.RenderMarkupToHTML(context.Background(), space.SystemSpace, content, markup.SystemMarkupMarkdown)
	// then
	require.Nil(s.T(), err)
	s.T().Log(result)
	require.NotNil(s.T(), result)
	expectedContent := "<p>Linking to non existing issue # in the content</p>\n"
	assert.Equal(s.T(), expectedContent, *result)
}

func (s *MarkupRenderingSuite) TestDoNotInsertLinkToNonNumericValue() {
	// when
	content := "Linking to non existing issue #FOO in the content"
	result, err := s.markupRenderer.RenderMarkupToHTML(context.Background(), space.SystemSpace, content, markup.SystemMarkupMarkdown)
	// then
	require.Nil(s.T(), err)
	s.T().Log(result)
	require.NotNil(s.T(), result)
	expectedContent := "<p>Linking to non existing issue #FOO in the content</p>\n"
	assert.Equal(s.T(), expectedContent, *result)
}
