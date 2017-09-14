package api

import (
	"net/http"
	"strings"

	"github.com/fabric8-services/fabric8-wit/api/authz"
	"github.com/fabric8-services/fabric8-wit/api/handler"
	"github.com/fabric8-services/fabric8-wit/api/model"
	"github.com/fabric8-services/fabric8-wit/auth"
	"github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/notification"
	"github.com/gin-gonic/gin"
)

// NewGinEngine instanciates a new HTTP engine to server the requests
func NewGinEngine(appDB *gormapplication.GormDB, notificationChannel notification.Channel, config *configuration.ConfigurationData) *gin.Engine {
	httpEngine := gin.Default()
	httpEngine.GET("/ping", func(c *gin.Context) {
		c.String(200, "pong")
	})
	authMiddleware := authz.NewJWTAuthMiddleware(appDB)
	spacesResource := handler.NewSpacesResource(appDB, config, auth.NewAuthzResourceManager(config))
	workitemsResource := handler.NewWorkItemsResource(appDB, notificationChannel, config)
	httpEngine.GET("/api/spaces/:spaceID", spacesResource.Show)
	httpEngine.GET("/api/spaces/:spaceID/workitems", workitemsResource.List)
	httpEngine.GET("/api/workitems/:workitemID", workitemsResource.Show)
	// secured endpoints
	authGroup := httpEngine.Group("/")
	authGroup.Use(authMiddleware.MiddlewareFunc())
	authGroup.GET("/api/spaces", spacesResource.List)
	authGroup.POST("/api/spaces/", spacesResource.Create)
	authGroup.POST("/api/spaces/:spaceID/workitems", workitemsResource.Create)
	authGroup.PATCH("/api/workitems/:workitemID", authz.NewWorkItemEditorAuthorizator(appDB, config), workitemsResource.Update)

	// register types for the JSON-API
	model.RegisterUUIDType()
	// If an /api/* route does not exist, redirect it to /legacyapi/* path
	// to be handled by goa
	httpEngine.NoRoute(func(ctx *gin.Context) {
		if strings.HasPrefix(ctx.Request.URL.Path, "/api/") {
			ctx.Redirect(http.StatusTemporaryRedirect, strings.Replace(ctx.Request.URL.Path, "/api/", "/legacyapi/", 1))
		} else {
			ctx.String(http.StatusNotFound, "Not found!")
		}
	})

	return httpEngine
}
