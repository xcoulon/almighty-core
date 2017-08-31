package api_test

import (
	"bytes"
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
		Context("Test API GET redirect", func() {

			Specify("Test API GET redirect", func() {
				// given
				r, _ := http.NewRequest(http.MethodGet, "/api/foobar", nil)

				// when
				rr := Execute(s.GinkgoTestSuite, r)

				// then
				assert.Equal(GinkgoT(), http.StatusTemporaryRedirect, rr.Code)
				assert.Equal(GinkgoT(), "/legacyapi/foobar", rr.HeaderMap.Get("Location"))
			})

		})
		Context("Test API POST redirect", func() {

			Specify("Test API POST redirect", func() {
				// given
				payload := bytes.NewBuffer(make([]byte, 0))
				r, _ := http.NewRequest(http.MethodPost, "/api/foobar", payload)

				// when
				rr := Execute(s.GinkgoTestSuite, r)

				// then
				assert.Equal(GinkgoT(), http.StatusTemporaryRedirect, rr.Code)
				assert.Equal(GinkgoT(), "/legacyapi/foobar", rr.HeaderMap.Get("Location"))
			})

		})
		Context("Test API PATCH redirect", func() {

			Specify("Test API PATCH redirect", func() {
				// given
				payload := bytes.NewBuffer(make([]byte, 0))
				r, _ := http.NewRequest(http.MethodPatch, "/api/foobar", payload)

				// when
				rr := Execute(s.GinkgoTestSuite, r)

				// then
				assert.Equal(GinkgoT(), http.StatusTemporaryRedirect, rr.Code)
				assert.Equal(GinkgoT(), "/legacyapi/foobar", rr.HeaderMap.Get("Location"))
			})

		})
		Context("Test API DELETE redirect", func() {

			Specify("Test API DELETE redirect", func() {
				// given
				r, _ := http.NewRequest(http.MethodDelete, "/api/foobar", nil)

				// when
				rr := Execute(s.GinkgoTestSuite, r)

				// then
				assert.Equal(GinkgoT(), http.StatusTemporaryRedirect, rr.Code)
				assert.Equal(GinkgoT(), "/legacyapi/foobar", rr.HeaderMap.Get("Location"))
			})

		})

	})
})
