package handler

import (
	"context"
	"strconv"
	"time"

	"github.com/google/jsonapi"
	uuid "github.com/satori/go.uuid"
	// jwt "github.com/dgrijalva/jwt-go"
	ginjwt "github.com/appleboy/gin-jwt"
	"github.com/fabric8-services/fabric8-wit/application"
	"github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/space/authz"
	"github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/gin-gonic/gin"
	errs "github.com/pkg/errors"
	jwt "gopkg.in/dgrijalva/jwt-go.v3"
)

//SigningKey the key for signing and verifying the JWT
var SigningKey []byte

func init() {
	config := configuration.Get()
	SigningKey = config.GetTokenPrivateKey()
}

// NewJWTAuthMiddleware initialises the JWT auth middleware
func NewJWTAuthMiddleware(db application.DB) *ginjwt.GinJWTMiddleware {
	config := configuration.Get()

	return &ginjwt.GinJWTMiddleware{
		Realm:      config.GetKeycloakRealm(),
		Key:        SigningKey, // will switch to public/private key once RSA256 is supported
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

//NewAuthorizatorHandler initializes the authorizator handler of the JWT Auth middleware
func NewAuthorizatorHandler(db application.DB) func(string, *gin.Context) bool {
	return func(userID string, ctx *gin.Context) bool {
		log.Debug(ctx, map[string]interface{}{"userID": userID}, "authorizing user...")
		err := db.Identities().CheckExists(ctx, userID)
		if err != nil {
			log.Error(ctx, map[string]interface{}{"userID": userID}, "user NOT authorized")
			return false
		}
		log.Debug(ctx, map[string]interface{}{"userID": userID}, "user authorized")
		ctx.Set(USER_ID, userID)
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

//WorkItemUpdateAuthorizator returns a new handler that checks if the current user is allowed to edit the work item
func WorkItemUpdateAuthorizator(db application.DB) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		workitemID, err := uuid.FromString(ctx.Param("workitemID")) // the workitem ID param
		if err != nil {
			abortWithError(ctx, err)
			return
		}
		wi, err := db.WorkItems().LoadByID(ctx, workitemID)
		if err != nil {
			abortWithError(ctx, err)
			return
		}
		creator := wi.Fields[workitem.SystemCreator]
		if creator == nil {
			abortWithError(ctx, errors.NewBadParameterError("workitem.SystemCreator", nil))
		}
		currentUserID, _ := GetUserID(ctx)
		if currentUserID == nil {
			abortWithError(ctx, errors.NewBadParameterError("workitem.SystemCreator", nil))
		}
		log.Debug(ctx, map[string]interface{}{"wi": wi, "creator": creator, "current_user": currentUserID}, "Authorizing work item update...")
		authorized, err := authorizeWorkitemEditor(ctx, db, wi.SpaceID, creator.(string), currentUserID.String())
		if err != nil {
			abortWithError(ctx, err)
			return
		}
		if !authorized {
			abortWithError(ctx, errors.NewForbiddenError("user is not allowed to update this work item"))
		}
	}
}

// Returns true if the user is the work item creator or space collaborator
func authorizeWorkitemEditor(ctx context.Context, db application.DB, spaceID uuid.UUID, creatorID string, editorID string) (bool, error) {
	if editorID == creatorID {
		return true, nil
	}
	authorized, err := authz.Authorize(ctx, spaceID.String())
	if err != nil {
		return false, errors.NewUnauthorizedError(err.Error())
	}
	return authorized, nil
}
