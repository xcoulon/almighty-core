package gormtestsupport

import (
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/gin-gonic/gin"

	"context"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/api"
	"github.com/fabric8-services/fabric8-wit/auth"
	"github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/login"
	"github.com/fabric8-services/fabric8-wit/migration"
	"github.com/fabric8-services/fabric8-wit/models"
	"github.com/fabric8-services/fabric8-wit/resource"
	"github.com/fabric8-services/fabric8-wit/token"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq" // need to import postgres driver
	. "github.com/onsi/ginkgo"
	errs "github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

var _ suite.SetupAllSuite = &DBTestSuite{}
var _ suite.TearDownAllSuite = &DBTestSuite{}

// NewGinkgoDBTestSuite instanciate a new DBTestSuite
func NewGinkgoDBTestSuite(configFilePath string) GinkgoDBTestSuite {
	s := GinkgoDBTestSuite{configFile: configFilePath}
	s.Setup()
	return s
}

// GinkgoDBTestSuite is a base for tests using Ginkgo with a gorm db
type GinkgoDBTestSuite struct {
	configFile    string
	Configuration *configuration.ConfigurationData
	TokenManager  token.Manager
	HTTPEngine    *gin.Engine
	DB            *gorm.DB
	Clean         func()
	testUser1     *TestUser
	testUser2     *TestUser
}

// TestUser the structure to hold data about the test user, including their access token retrieved from KC.
type TestUser struct {
	User        account.User
	Identity    account.Identity
	AccessToken string
}

// Setup initializes the DB connection
func (s *GinkgoDBTestSuite) Setup() {
	// resource.Require(s.T(), resource.Database)
	config, err := configuration.NewConfigurationData(s.configFile)
	if err != nil {
		log.Panic(nil, map[string]interface{}{
			"err": err,
		}, "failed to setup the configuration")
	}
	s.Configuration = config
	if _, c := os.LookupEnv(resource.Database); c != false {
		s.DB, err = gorm.Open("postgres", s.Configuration.GetPostgresConfigString())
		if err != nil {
			log.Panic(nil, map[string]interface{}{
				"err":             err,
				"postgres_config": config.GetPostgresConfigString(),
			}, "failed to connect to the database")
		}
	}
	s.TokenManager, err = token.NewManager(s.Configuration)
	if err != nil {
		log.Panic(nil, map[string]interface{}{
			"err": err,
		}, "failed to setup the token manager")
	}
	s.HTTPEngine = api.NewGinEngine(gormapplication.NewGormDB(s.DB), nil, s.TokenManager, s.Configuration, nil)

}

// PopulateGinkgoDBTestSuite populates the DB with common values
func (s *GinkgoDBTestSuite) PopulateGinkgoDBTestSuite(ctx context.Context) {
	if _, c := os.LookupEnv(resource.Database); c != false {
		if err := models.Transactional(s.DB, func(tx *gorm.DB) error {
			return migration.PopulateCommonTypes(ctx, tx, workitem.NewWorkItemTypeRepository(tx))
		}); err != nil {
			log.Panic(nil, map[string]interface{}{
				"err":             err,
				"postgres_config": s.Configuration.GetPostgresConfigString(),
			}, "failed to populate the database with common types")
		}
	}
}

// TearDown closes the DB connection
func (s *GinkgoDBTestSuite) TearDown() {
	s.DB.Close()
}

// DisableGormCallbacks will turn off gorm's automatic setting of `created_at`
// and `updated_at` columns. Call this function and make sure to `defer` the
// returned function.
//
//    resetFn := DisableGormCallbacks()
//    defer resetFn()
func (s *GinkgoDBTestSuite) DisableGormCallbacks() func() {
	gormCallbackName := "gorm:update_time_stamp"
	// remember old callbacks
	oldCreateCallback := s.DB.Callback().Create().Get(gormCallbackName)
	oldUpdateCallback := s.DB.Callback().Update().Get(gormCallbackName)
	// remove current callbacks
	s.DB.Callback().Create().Remove(gormCallbackName)
	s.DB.Callback().Update().Remove(gormCallbackName)
	// return a function to restore old callbacks
	return func() {
		s.DB.Callback().Create().Register(gormCallbackName, oldCreateCallback)
		s.DB.Callback().Update().Register(gormCallbackName, oldUpdateCallback)
	}
}

//TestUser1 return the TestUser structure for the testuser1
func (s *GinkgoDBTestSuite) TestUser1() *TestUser {
	if s.testUser1 == nil {
		s.testUser1 = s.generateTestUserIdentityAndToken(s.Configuration.GetKeycloakTestUserName(), s.Configuration.GetKeycloakTestUserSecret())
	}
	return s.testUser1
}

//TestUser2 return the TestUser structure for the testuser2
func (s *GinkgoDBTestSuite) TestUser2() *TestUser {
	if s.testUser2 == nil {
		s.testUser2 = s.generateTestUserIdentityAndToken(s.Configuration.GetKeycloakTestUser2Name(), s.Configuration.GetKeycloakTestUser2Secret())
	}
	return s.testUser2
}

// GenerateTestUserIdentityAndToken calls the KC instance to retrieve the token for the given username and secret and generates the user/identity records in the DB
func (s *GinkgoDBTestSuite) generateTestUserIdentityAndToken(username, userSecret string) *TestUser {
	identityRepository := account.NewIdentityRepository(s.DB)
	userRepository := account.NewUserRepository(s.DB)
	appDB := gormapplication.NewGormDB(s.DB)

	auth := login.NewKeycloakOAuthProvider(identityRepository, userRepository, s.TokenManager, appDB)
	tokenEndpoint, err := s.Configuration.GetKeycloakEndpointToken(&http.Request{Host: "api.example.org"})
	require.Nil(GinkgoT(), err, "unable to get Keycloak token endpoint URL")
	accessToken, err := s.generateAccessToken(tokenEndpoint, username, userSecret)
	require.Nil(GinkgoT(), err, "unable to generate test token")
	// Creates the testuser user and identity if they don't yet exist
	profileEndpoint, err := s.Configuration.GetKeycloakAccountEndpoint(&http.Request{Host: "api.example.org"})
	require.Nil(GinkgoT(), err, "unable to get Keycloak account endpoint URL")
	identity, user, err := auth.CreateOrUpdateKeycloakUser(*accessToken, context.Background(), profileEndpoint)
	require.Nil(GinkgoT(), err, "unable to create or update keycloak user and identity")
	return &TestUser{
		Identity:    *identity,
		User:        *user,
		AccessToken: *accessToken,
	}
}

func (s *GinkgoDBTestSuite) generateAccessToken(tokenEndpoint, username, userSecret string) (*string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	res, err := client.PostForm(tokenEndpoint, url.Values{
		"client_id":     {s.Configuration.GetKeycloakClientID()},
		"client_secret": {s.Configuration.GetKeycloakSecret()},
		"username":      {username},
		"password":      {userSecret},
		"grant_type":    {"password"},
	})
	if err != nil {
		return nil, errors.NewInternalError(context.Background(), errs.Wrap(err, "error when obtaining token"))
	}
	token, err := auth.ReadToken(context.Background(), res)
	require.Nil(GinkgoT(), err)
	accessToken := token.AccessToken
	log.Info(nil, map[string]interface{}{"username": username}, "Token: %s", *accessToken)
	_, err = jwt.Parse(*accessToken, func(t *jwt.Token) (interface{}, error) {
		return s.TokenManager.DevModePublicKey(), nil
	})
	if err != nil {
		log.Error(nil, map[string]interface{}{
			"err": err,
		}, "unable to parse access token after we got it (wrong public key ?)")
		return nil, err
	}

	return accessToken, nil
}
