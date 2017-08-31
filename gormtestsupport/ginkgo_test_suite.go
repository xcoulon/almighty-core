package gormtestsupport

import (
	"net/http"
	"net/url"
	"os"
	"time"

	"context"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/fabric8-services/fabric8-wit/account"
	"github.com/fabric8-services/fabric8-wit/auth"
	"github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/gormsupport"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/login"
	"github.com/fabric8-services/fabric8-wit/migration"
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

// NewGinkgoTestSuite instanciate a new DBTestSuite
func NewGinkgoTestSuite(configFilePath string) GinkgoTestSuite {
	s := GinkgoTestSuite{configFile: configFilePath}
	s.Setup()
	return s
}

// GinkgoTestSuite is a base for tests using Ginkgo with a gorm db
type GinkgoTestSuite struct {
	configFile    string
	Configuration *configuration.ConfigurationData
	DB            *gorm.DB
	Clean         func()
}

// Setup initializes the DB connection
func (s *GinkgoTestSuite) Setup() {
	// resource.Require(s.T(), resource.Database)
	config, err := configuration.NewConfigurationData(s.configFile, true)
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
}

// PopulateGinkgoTestSuite populates the DB with common values
func (s *GinkgoTestSuite) PopulateGinkgoTestSuite(ctx context.Context) {
	if _, c := os.LookupEnv(resource.Database); c != false {
		if err := gormsupport.Transactional(s.DB, func(tx *gorm.DB) error {
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
func (s *GinkgoTestSuite) TearDown() {
	s.DB.Close()
}

// DisableGormCallbacks will turn off gorm's automatic setting of `created_at`
// and `updated_at` columns. Call this function and make sure to `defer` the
// returned function.
//
//    resetFn := DisableGormCallbacks()
//    defer resetFn()
func (s *GinkgoTestSuite) DisableGormCallbacks() func() {
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

// GenerateTestUserIdentityAndToken calls the KC instance to retrieve the token for the given username and secret and generates the user/identity records in the DB
func (s *GinkgoTestSuite) GenerateTestUserIdentityAndToken(username, userSecret string) (*account.Identity, *account.User, *string, error) {
	identityRepository := account.NewIdentityRepository(s.DB)
	userRepository := account.NewUserRepository(s.DB)
	appDB := gormapplication.NewGormDB(s.DB)
	publicKey, err := s.Configuration.GetTokenPublicKey()
	if err != nil {
		log.Panic(nil, map[string]interface{}{
			"err": err,
		}, "failed to parse public token")
	}
	tokenManager := token.NewManager(publicKey)

	auth := login.NewKeycloakOAuthProvider(identityRepository, userRepository, tokenManager, appDB)
	tokenEndpoint, err := s.Configuration.GetKeycloakEndpointToken(&http.Request{Host: "api.example.org"})
	if err != nil {
		log.Error(nil, map[string]interface{}{
			"err": err,
		}, "unable to get Keycloak token endpoint URL")
		return nil, nil, nil, errs.Wrap(err, "unable to get Keycloak token endpoint URL")
	}
	log.Warn(nil, map[string]interface{}{"tokenEndpoint": tokenEndpoint}, "Retrieved token Endpoint")

	accessToken, err := s.generateAccessToken(tokenEndpoint, username, userSecret)
	if err != nil {
		log.Error(nil, map[string]interface{}{
			"err":      err,
			"username": username,
		}, "unable to get Generate User token")
		return nil, nil, nil, errs.Wrap(err, "unable to generate test token ")
	}

	// Creates the testuser user and identity if they don't yet exist
	profileEndpoint, err := s.Configuration.GetKeycloakAccountEndpoint(&http.Request{Host: "api.example.org"})
	if err != nil {
		log.Error(nil, map[string]interface{}{
			"err": err,
		}, "unable to get Keycloak account endpoint URL")
		return nil, nil, nil, err
	}
	identity, user, err := auth.CreateOrUpdateKeycloakUser(*accessToken, context.Background(), profileEndpoint)
	if err != nil {
		log.Error(nil, map[string]interface{}{
			"err": err,
		}, "unable to create or update keycloak user and identity")
		return nil, nil, nil, err
	}
	return identity, user, accessToken, nil
}

func (s *GinkgoTestSuite) generateAccessToken(tokenEndpoint, username, userSecret string) (*string, error) {
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
	log.Debug(nil, nil, "Token: %s", *accessToken)
	_, err = jwt.Parse(*accessToken, func(t *jwt.Token) (interface{}, error) {
		return s.Configuration.GetTokenPublicKey()
	})
	if err != nil {
		log.Error(nil, map[string]interface{}{
			"err": err,
		}, "unable to parse access token after we got it")
		return nil, err
	}

	log.Info(nil, nil, "access token retrieved and validated :)")
	return accessToken, nil
}
