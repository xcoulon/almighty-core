package authz

import (
	"crypto/rsa"
	"strconv"
	"time"

	"github.com/google/jsonapi"
	uuid "github.com/satori/go.uuid"
	// jwt "github.com/dgrijalva/jwt-go"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/gin-gonic/gin"
	errs "github.com/pkg/errors"
	ginjwt "github.com/xcoulon/gin-jwt"
	"gopkg.in/dgrijalva/jwt-go.v3"
)

//SigningKey the key for signing the JWT
var SigningKey *rsa.PrivateKey

//VerifyKey the key for verifying the JWT
var VerifyKey *rsa.PublicKey

func init() {
	var err error
	SigningKey, err = configuration.Get().GetTokenPrivateKey()
	if err != nil {
		log.Panic(nil, nil, "Unable to load private key: ", err.Error())
	}
	VerifyKey, err = configuration.Get().GetTokenPublicKey()
	if err != nil {
		log.Panic(nil, nil, "Unable to load public key: ", err.Error())
	}
}

// NewJWTAuthMiddleware initialises the JWT auth middleware
func NewJWTAuthMiddleware(db application.DB) *ginjwt.GinJWTMiddleware {
	config := configuration.Get()
	return &ginjwt.GinJWTMiddleware{
		Realm: config.GetKeycloakRealm(),
		// SignKey:    SigningKey, // will switch to public/private key once RSA256 is supported
		VerifyKey:  VerifyKey, // will switch to public/private key once RSA256 is supported
		Timeout:    time.Hour,
		MaxRefresh: time.Hour,
		// signing algorithm - possible values are HS256, HS384, HS512
		// asymetric algorithms not supported yet. See https://github.com/appleboy/gin-jwt/pull/80
		SigningAlgorithm: "RS256",
		IdentityHandler:  NewIdentityHandler(),
		// Authorizator:     NewAuthorizator(),
		// Callback function that will be called during login.
		// Using this function it is possible to add additional payload data to the webtoken.
		// The data is then made available during requests via c.Get("JWT_PAYLOAD").
		// Note that the payload is not encrypted.
		// The attributes mentioned on jwt.io can't be used as keys for the map.
		// Optional, by default no additional data will be set.
		PayloadFunc: NewPayloadFunc(),

		// Authorizator: NewAuthorizatorHandler(db),
		Unauthorized: NewUnauthorizedHandler(),
		// TokenLookup is a string in the form of "<source>:<name>" that is used
		// to extract token from the request.
		// Optional. Default value "header:Authorization".
		// Possible values:
		// - "header:<name>"
		// - "query:<name>"
		// - "cookie:<name>"
		TokenLookup: "header:Authorization",
		// TokenLookup: "query:token",
		// TokenLookup: "cookie:token",

		// TokenHeadName is a string in the header. Default value is "Bearer"
		TokenHeadName: "Bearer",

		// TimeFunc provides the current time. You can override it to use another time value. This is useful for testing or if your server uses a different time zone than your tokens.
		TimeFunc: time.Now,
	}
}

const (
	SUBJECT_CLAIM string = "sub"
	USER_ID       string = "user_id"
)

// NewIdentityHandler returns a new identity handler that will look for the `subject` claim in the JWT
func NewIdentityHandler() func(jwt.MapClaims) string {
	return func(claims jwt.MapClaims) string {
		if subject, ok := claims[SUBJECT_CLAIM]; ok {
			return subject.(string)
		}
		log.Warn(nil, nil, "JWT did not contain any `sub` (subject) claim.")
		return ""
	}
}

// NewAuthorizationHandler initializes the authorizator handler of the JWT Auth middleware
// This handler checks that the given userID corresponds to a valid identity in the DB.
func NewAuthorizationHandler(db application.DB) func(string, *gin.Context) bool {
	return func(userID string, ctx *gin.Context) bool {
		log.Info(ctx, map[string]interface{}{"userID": userID}, "authenticating user...")
		err := db.Identities().CheckExists(ctx, userID)
		if err != nil {
			log.Error(ctx, map[string]interface{}{"userID": userID}, "user NOT authorized")
			return false
		}
		log.Info(ctx, map[string]interface{}{"userID": userID}, "user authorized")
		ctx.Set(USER_ID, userID)
		return true
	}
}

// NewPayloadFunc is a callback function that will be called during login.
// Using this function it is possible to add additional payload data to the webtoken.
// The data is then made available during requests via c.Get("JWT_PAYLOAD").
// Note that the payload is not encrypted.
// The attributes mentioned on jwt.io can't be used as keys for the map.
func NewPayloadFunc() func(userID string) map[string]interface{} {
	return func(userID string) map[string]interface{} {
		// TODO: during login, request the authorizations from KC for the given user
		return nil
	}
}

// NewAuthorizatorHandler initializes the authorizator handler of the JWT Auth middleware
// func NewAuthorizatorHandler(db application.DB) func(string, *gin.Context) bool {
// 	return func(userID string, ctx *gin.Context) bool {
// 		log.Debug(ctx, map[string]interface{}{"userID": userID}, "authorizing user...")
// 		err := db.Identities().CheckExists(ctx, userID)
// 		if err != nil {
// 			log.Error(ctx, map[string]interface{}{"userID": userID}, "user NOT authorized")
// 			return false
// 		}
// 		log.Debug(ctx, map[string]interface{}{"userID": userID}, "user authorized")
// 		ctx.Set(USER_ID, userID)
// 		return true
// 	}
// }

//NewUnauthorizedHandler initializes the unauthorized handler of the JWT Auth middleware
func NewUnauthorizedHandler() func(*gin.Context, int, string) {
	return func(ctx *gin.Context, status int, message string) {
		log.Info(ctx, map[string]interface{}{"status": status, "message": message}, "user not unauthorized")
		ctx.Status(status)
		ctx.Header("Content-Type", jsonapi.MediaType)
		jsonapi.MarshalErrors(ctx.Writer, []*jsonapi.ErrorObject{{
			Status: strconv.Itoa(status),
			Meta:   &map[string]interface{}{"error": message},
		}})
	}
}

//GetUserID returns the ID of the current user as a UUID or an error if something wrong happened (key not found or not an UUID)
func GetUserID(ctx *gin.Context) (*uuid.UUID, error) {
	if ctxValue, ok := ctx.Get(USER_ID); ok {
		if ctxValueStr, ok := ctxValue.(string); ok {
			userID, err := uuid.FromString(ctxValueStr)
			if err != nil {
				return nil, errs.Wrapf(err, "failed to parse user id value '%s' as a UUID", ctxValueStr)
			}
			return &userID, nil
		}
		return nil, errs.Errorf("request context did not contain a valid UUID entry for the current user's ID: '%s'", ctxValue)
	}
	return nil, errs.New("request context did not contain an entry for the current user's ID")
}
