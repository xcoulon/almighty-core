package controller_test

import (
	"context"
	"testing"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	. "github.com/almighty/almighty-core/controller"
	"github.com/almighty/almighty-core/gormapplication"
	"github.com/almighty/almighty-core/gormsupport/cleaner"
	"github.com/almighty/almighty-core/gormtestsupport"
	"github.com/almighty/almighty-core/markup"
	"github.com/almighty/almighty-core/migration"
	"github.com/almighty/almighty-core/resource"
	"github.com/goadesign/goa"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// a normal test function that will kick off MarkupRenderingSuite
func TestSuiteMarkupRendering(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, new(MarkupRenderingSuite))
}

// ========== MarkupRenderingSuite struct that implements SetupSuite, TearDownSuite, SetupTest, TearDownTest ==========
type MarkupRenderingSuite struct {
	gormtestsupport.DBTestSuite
	db         *gormapplication.GormDB
	controller app.RenderController
	clean      func()

	svc *goa.Service
}

func (s *MarkupRenderingSuite) SetupSuite() {
	s.DBTestSuite.SetupSuite()
	s.DBTestSuite.PopulateDBTestSuite(migration.NewMigrationContext(context.Background()))
}

func (s *MarkupRenderingSuite) SetupTest() {
	s.db = gormapplication.NewGormDB(s.DB)
	s.svc = goa.New("Rendering-service-test")
	s.controller = NewRenderController(s.svc, s.db)
	s.clean = cleaner.DeleteCreatedEntities(s.DB)
}

func (s *MarkupRenderingSuite) TearDownTest() {
	s.clean()
}

func (s *MarkupRenderingSuite) TestRenderPlainText() {
	// given
	payload := app.MarkupRenderingPayload{Data: &app.MarkupRenderingPayloadData{
		Type: RenderingType,
		Attributes: &app.MarkupRenderingPayloadDataAttributes{
			Content: "foo",
			Markup:  markup.SystemMarkupPlainText,
		}}}
	// when
	_, result := test.RenderRenderOK(s.T(), s.svc.Context, s.svc, s.controller, &payload)
	// then
	require.NotNil(s.T(), result)
	require.NotNil(s.T(), result.Data)
	assert.Equal(s.T(), "foo", result.Data.Attributes.RenderedContent)
}

func (s *MarkupRenderingSuite) TestRenderMarkdown() {
	// given
	payload := app.MarkupRenderingPayload{Data: &app.MarkupRenderingPayloadData{
		Type: RenderingType,
		Attributes: &app.MarkupRenderingPayloadDataAttributes{
			Content: "foo",
			Markup:  markup.SystemMarkupMarkdown,
		}}}

	// when
	_, result := test.RenderRenderOK(s.T(), s.svc.Context, s.svc, s.controller, &payload)
	// then
	require.NotNil(s.T(), result)
	require.NotNil(s.T(), result.Data)
	assert.Equal(s.T(), "<p>foo</p>\n", result.Data.Attributes.RenderedContent)
}

func (s *MarkupRenderingSuite) TestRenderUnsupportedMarkup() {
	// given
	payload := app.MarkupRenderingPayload{Data: &app.MarkupRenderingPayloadData{
		Type: RenderingType,
		Attributes: &app.MarkupRenderingPayloadDataAttributes{
			Content: "foo",
			Markup:  "bar",
		}}}

	// when/then
	test.RenderRenderBadRequest(s.T(), s.svc.Context, s.svc, s.controller, &payload)
}
