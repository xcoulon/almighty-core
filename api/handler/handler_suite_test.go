package handler_test

import (
	"testing"

	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/gormsupport/cleaner"
	"github.com/fabric8-services/fabric8-wit/gormtestsupport"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/require"
)

func TestHandler(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Handler Suite")
}

type TestUser struct {
	User        account.User
	Identity    account.Identity
	AccessToken string
}

var testSuite gormtestsupport.GinkgoTestSuite
var testuser1 TestUser
var testuser2 TestUser

// register `testuser` and `testuser2` in the DB and retrieve access tokens in KC for those 2 users
var _ = BeforeSuite(func() {
	testSuite = gormtestsupport.NewGinkgoTestSuite("../../config.yaml")
	testSuite.Clean = cleaner.DeleteCreatedEntities(testSuite.DB)
	testusers := map[string]string{
		testSuite.Configuration.GetKeycloakTestUserName():  testSuite.Configuration.GetKeycloakTestUserSecret(),
		testSuite.Configuration.GetKeycloakTestUser2Name(): testSuite.Configuration.GetKeycloakTestUser2Secret()}
	for name, secret := range testusers {
		identity, user, accessToken, err := testSuite.GenerateTestUserIdentityAndToken(name, secret)
		require.Nil(GinkgoT(), err, "error while creating test user '%s'", name)
		GinkgoT().Logf("Generated access token: %s\n", *accessToken)
		testuser1 = TestUser{
			AccessToken: *accessToken,
			Identity:    *identity,
			User:        *user,
		}
	}
})
var _ = AfterSuite(func() {
	testSuite.Clean()
})
