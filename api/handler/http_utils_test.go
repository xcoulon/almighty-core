package handler_test

import (
	"net/http"
	"net/http/httptest"

	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
)

// Execute submits the request and returns the response recording fo subseauent verifications
func Execute(s gormtestsupport.GinkgoDBTestSuite, r *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	s.HTTPEngine.ServeHTTP(rr, r)
	return rr
}
