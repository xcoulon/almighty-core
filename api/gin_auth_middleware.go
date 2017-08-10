package api

import (
	"strconv"
	"time"

	"github.com/google/jsonapi"
	// jwt "github.com/dgrijalva/jwt-go"
	ginjwt "github.com/appleboy/gin-jwt"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/gin-gonic/gin"
	jwt "gopkg.in/dgrijalva/jwt-go.v3"
)

//Key the key for signing and verifying the JWT
var Key []byte

func init() {
	config := configuration.Get()
	Key = config.GetTokenPrivateKey()
}

// NewJWTAuthMiddleware initialises the JWT auth middleware
func NewJWTAuthMiddleware(db application.DB) *ginjwt.GinJWTMiddleware {
	config := configuration.Get()

	return &ginjwt.GinJWTMiddleware{
		Realm:      config.GetKeycloakRealm(),
		Key:        Key, // will switch to public/private key once RSA256 is supported
		Timeout:    time.Hour,
		MaxRefresh: time.Hour,
		// signing algorithm - possible values are HS256, HS384, HS512
		// asymetric algorithms not supported yet. See https://github.com/appleboy/gin-jwt/pull/80
		SigningAlgorithm: "HS256",
		// Authenticator: func(userId string, password string, ctx *gin.Context) (string, bool) {
		// 	log.Info(ctx, map[string]interface{}{"userID": userId}, "Authenticating user...")
		// 	if (userId == "admin" && password == "admin") || (userId == "test" && password == "test") {
		// 		log.Info(ctx, map[string]interface{}{"userID": userId}, "user authenticated")
		// 		return userId, true
		// 	}
		// 	log.Warn(ctx, map[string]interface{}{"userID": userId}, "user not authenticated")
		// 	return userId, false
		// },
		IdentityHandler: NewIdentityHandler(),
		// Authenticator:   NewAuthenticationHandler(db),
		Authorizator: NewAuthorizatorHandler(db),
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

// NewIdentityHandler returns a new identity handler that will look for the `subject` claim in the JWT
func NewIdentityHandler() func(jwt.MapClaims) string {
	return func(claims jwt.MapClaims) string {
		if subject, ok := claims["sub"]; ok {
			return subject.(string)
		}
		log.Warn(nil, nil, "JWT did not contain any `sub` claim.")
		return ""
	}
}

//NewAuthenticationHandler initializes the authorizator handler of the JWT Auth middleware
// func NewAuthenticationHandler(db application.DB) func(string, *gin.Context) bool {
// 	return func(userID string, ctx *gin.Context) bool {
// 		log.Info(ctx, map[string]interface{}{"userID": userID}, "authorizing user...")
// 		err := db.Identities().CheckExists(ctx, userID)
// 		if err != nil {
// 			log.Error(ctx, map[string]interface{}{"userID": userID}, "user NOT authorized")
// 			return false
// 		}
// 		log.Debug(ctx, map[string]interface{}{"userID": userID}, "user authorized")
// 		return true
// 	}
// }

//NewAuthorizatorHandler initializes the authorizator handler of the JWT Auth middleware
func NewAuthorizatorHandler(db application.DB) func(string, *gin.Context) bool {
	return func(userID string, ctx *gin.Context) bool {
		log.Info(ctx, map[string]interface{}{"userID": userID}, "authorizing user...")
		err := db.Identities().CheckExists(ctx, userID)
		if err != nil {
			log.Error(ctx, map[string]interface{}{"userID": userID}, "user NOT authorized")
			return false
		}
		log.Debug(ctx, map[string]interface{}{"userID": userID}, "user authorized")
		return true
	}
}

//NewUnauthorizedHandler initializes the unauthorized handler of the JWT Auth middleware
func NewUnauthorizedHandler() func(*gin.Context, int, string) {
	return func(ctx *gin.Context, status int, message string) {
		log.Info(ctx, nil, "user not unauthorized")
		ctx.Status(status)
		ctx.Header("Content-Type", jsonapi.MediaType)
		jsonapi.MarshalErrors(ctx.Writer, []*jsonapi.ErrorObject{{
			Status: strconv.Itoa(status),
			Meta:   &map[string]interface{}{"error": message},
		}})
	}
}
