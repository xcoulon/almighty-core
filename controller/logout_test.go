package controller

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/app/test"
	"github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/fabric8-services/fabric8-wit/login"
	"github.com/fabric8-services/fabric8-wit/resource"
	testsupport "github.com/fabric8-services/fabric8-wit/test"
	wittoken "github.com/fabric8-services/fabric8-wit/token"

	"github.com/goadesign/goa"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type TestLogoutREST struct {
	suite.Suite
	config *configuration.ConfigurationData
}

func TestRunLogoutREST(t *testing.T) {
	resource.Require(t, resource.UnitTest)
	config := configuration.LoadDefault()
	suite.Run(t, &TestLogoutREST{config: config})
}

func (rest *TestLogoutREST) SetupTest() {
}

func (rest *TestLogoutREST) TearDownTest() {
}

func (rest *TestLogoutREST) UnSecuredController() (*goa.Service, *LogoutController) {
	priv, _ := wittoken.RSAPrivateKey()

	svc := testsupport.ServiceAsUser("Logout-Service", wittoken.NewManagerWithPrivateKey(priv), testsupport.TestIdentity)
	return svc, &LogoutController{Controller: svc.NewController("logout"), logoutService: &login.KeycloakLogoutService{}, configuration: rest.config}
}

func (rest *TestLogoutREST) TestLogoutRedirects() {
	t := rest.T()
	resource.Require(t, resource.UnitTest)
	svc, ctrl := rest.UnSecuredController()

	redirect := "http://domain.com"
	resp := test.LogoutLogoutTemporaryRedirect(t, svc.Context, svc, ctrl, &redirect)
	assert.Equal(t, resp.Header().Get("Cache-Control"), "no-cache")
}

func (rest *TestLogoutREST) TestLogoutWithoutReffererAndRedirectParamsBadRequest() {
	t := rest.T()
	resource.Require(t, resource.UnitTest)
	svc, ctrl := rest.UnSecuredController()

	test.LogoutLogoutBadRequest(t, svc.Context, svc, ctrl, nil)
}
