package resource_test

import (
	"net/http/httptest"
	"testing"

	"github.com/fabric8-services/fabric8-wit/api"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
)

func makeTokenString(SigningAlgorithm string, identity string) string {
	if SigningAlgorithm == "" {
		SigningAlgorithm = "HS256"
	}
	token := jwt.New(jwt.GetSigningMethod(SigningAlgorithm))
	claims := token.Claims.(jwt.MapClaims)
	claims["sub"] = identity // subject
	claims["exp"] = time.Now().Add(time.Hour).Unix()
	claims["orig_iat"] = time.Now().Unix()
	// config := configuration.Get()
	tss, _ := token.SigningString()
	fmt.Printf("Submitted signing string: '%v'\n", tss)
	tokenString, _ := token.SignedString(api.Key)
	return tokenString
}

func verify(t *testing.T, expectedStatusCode int) {
	rr := httptest.NewRecorder()
	httpEngine := api.NewGinEngine(gormapplication.NewGormDB(s.DB), s.Configuration)
	httpEngine.ServeHTTP(rr, r)

	if e, a := expectedStatusCode, rr.Code; e != a {
		s.T().Fatalf("Expected a status of %d, got %d", e, a)
	}

}
