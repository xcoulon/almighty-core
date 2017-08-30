package test

import (
	"net/http"
	"net/http/httptest"

	"github.com/fabric8-services/fabric8-wit/api"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	. "github.com/onsi/ginkgo"
)

// Execute submits the request and returns the response recording fo subseauent verifications
func Execute(s gormtestsupport.GinkgoTestSuite, r *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	httpEngine := api.NewGinEngine(gormapplication.NewGormDB(s.DB), nil, s.Configuration)
	httpEngine.ServeHTTP(rr, r)
	GinkgoT().Logf("Response status: %d", rr.Code)
	return rr
}
