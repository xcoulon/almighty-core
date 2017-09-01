package handler_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/fabric8-services/fabric8-wit/api"
	"github.com/fabric8-services/fabric8-wit/api/authz"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	. "github.com/onsi/ginkgo"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
)

func makeTokenString(SigningAlgorithm string, identity string, editableSpaces []uuid.UUID) string {
	token := jwt.New(jwt.GetSigningMethod(SigningAlgorithm))
	claims := token.Claims.(jwt.MapClaims)
	claims["sub"] = identity // subject
	claims["exp"] = time.Now().Add(time.Hour).Unix()
	claims["orig_iat"] = time.Now().Unix()

	tss, _ := token.SigningString()
	fmt.Printf("Submitted signing string: '%v'\n", tss)
	tokenString, err := token.SignedString(authz.SigningKey)
	require.Nil(GinkgoT(), err, "error while signing the token")
	return tokenString
}

// Execute submits the request and returns the response recording fo subseauent verifications
func Execute(s gormtestsupport.GinkgoTestSuite, r *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	// TODO: see how to move this HTTP engine setup in the BeforeSuite() function
	httpEngine := api.NewGinEngine(gormapplication.NewGormDB(s.DB), nil, s.Configuration)
	httpEngine.ServeHTTP(rr, r)
	GinkgoT().Logf("Response status: %d", rr.Code)
	return rr
}

func verify(s gormtestsupport.GinkgoTestSuite, r *http.Request, expectedStatusCode int) {
	rr := httptest.NewRecorder()
	httpEngine := api.NewGinEngine(gormapplication.NewGormDB(s.DB), nil, s.Configuration)
	httpEngine.ServeHTTP(rr, r)
	GinkgoT().Logf("Response:\n%s", rr.Body.String())
	if e, a := expectedStatusCode, rr.Code; e != a {
		GinkgoT().Fatalf("Expected a status of %d, got %d", e, a)
	}
}
