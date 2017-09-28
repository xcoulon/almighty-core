package api

import (
	"net/http"
	"regexp"
	"strings"
	"time"

	apiauth "github.com/fabric8-services/fabric8-wit/api/auth"
	"github.com/fabric8-services/fabric8-wit/api/handler"
	"github.com/fabric8-services/fabric8-wit/api/model"
	"github.com/fabric8-services/fabric8-wit/auth"
	"github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/notification"
	"github.com/fabric8-services/fabric8-wit/token"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// NewGinEngine instanciates a new HTTP engine to server the requests
func NewGinEngine(appDB *gormapplication.GormDB, notificationChannel notification.Channel, tokenManager token.Manager, config *configuration.ConfigurationData, redirectToGoa func(*gin.Context)) *gin.Engine {
	httpEngine := gin.Default()
	// CORS for https://foo.com and https://github.com origins, allowing:
	// - PUT and PATCH methods
	// - Origin header
	// - Credentials share
	// - Preflight requests cached for 12 hours
	httpEngine.Use(cors.New(cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
		AllowHeaders:     []string{"X-Request-Id", "Content-Type", "Authorization", "If-None-Match", "If-Modified-Since"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		// AllowOrigins:     []string{".*openshift.io", "localhost"},
		AllowOriginFunc: allowOriginsFunc("[.*openshift.io|localhost]"),
		MaxAge:          600 * time.Second,
	}))
	httpEngine.GET("/ping", func(c *gin.Context) {
		c.String(200, "pong")
	})
	spacesResource := handler.NewSpacesResource(appDB, config, auth.NewAuthzResourceManager(config))
	namedspacesResource := handler.NewNamedspacesResource(appDB)
	workitemsResource := handler.NewWorkItemsResource(appDB, notificationChannel, config)
	httpEngine.GET("/api/spaces/:spaceID", spacesResource.Show)
	httpEngine.GET("/api/spaces/:spaceID/workitems", workitemsResource.List)
	httpEngine.GET("/api/workitems/:workitemID", workitemsResource.Show)
	httpEngine.GET("/api/namedspaces/:userName", namedspacesResource.List)
	httpEngine.GET("/api/namedspaces/:userName/:spaceName", namedspacesResource.Show)
	// secured endpoints
	authMiddleware := apiauth.NewJWTAuthMiddleware(appDB, config.GetKeycloakRealm(), tokenManager.DevModePublicKey())
	authGroup := httpEngine.Group("/")
	authGroup.Use(authMiddleware.MiddlewareFunc())
	authGroup.GET("/api/spaces", spacesResource.List)
	authGroup.POST("/api/spaces", spacesResource.Create)
	authGroup.POST("/api/spaces/:spaceID/workitems", workitemsResource.Create)
	authGroup.PATCH("/api/workitems/:workitemID", apiauth.NewWorkItemEditorAuthorizator(appDB, config), workitemsResource.Update)

	// register types for the JSON-API
	model.RegisterUUIDType()
	// If an /api/* route does not exist, redirect it to /legacyapi/* path
	// to be handled by goa
	httpEngine.NoRoute(func(ctx *gin.Context) {
		if strings.HasPrefix(ctx.Request.URL.Path, "/api/") {
			redirectToGoa(ctx)
		} else {
			ctx.String(http.StatusNotFound, "Not found!")
		}
	})

	return httpEngine
}

func allowOriginsFunc(pattern string) func(origin string) bool {
	r := regexp.MustCompile(pattern)
	return func(origin string) bool {
		return r.Match([]byte(origin))
	}
}
