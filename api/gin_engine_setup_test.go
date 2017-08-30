package api_test

import (
	"context"
	"net/http"

	. "github.com/onsi/ginkgo"
	"github.com/stretchr/testify/assert"

	. "github.com/fabric8-services/fabric8-wit/api/test"
	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
)

type GinEngineTestSuite struct {
	gormtestsupport.GinkgoTestSuite
	clean func()
	ctx   context.Context
}

var _ = Describe("Engine", func() {

	s := GinEngineTestSuite{GinkgoTestSuite: gormtestsupport.NewGinkgoTestSuite("../config.yaml")}

	BeforeSuite(func() {
		s.Setup()
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

	Describe("Test API redirect", func() {
		Context("Test API redirect", func() {

			Specify("Test API redirect", func() {
				// given
				r, _ := http.NewRequest(http.MethodGet, "/api/foobar", nil)

				// when
				rr := Execute(s.GinkgoTestSuite, r)

				// then
				assert.Equal(GinkgoT(), http.StatusTemporaryRedirect, rr.Code)
				assert.Equal(GinkgoT(), "/legacyapi/foobar", rr.HeaderMap.Get("Location"))
			})

		})

	})
})
