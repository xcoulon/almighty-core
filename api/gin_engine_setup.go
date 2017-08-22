package api

import (
	"github.com/fabric8-services/fabric8-wit/api/authz"
	"github.com/fabric8-services/fabric8-wit/api/handler"
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
	spacesResource := handler.NewSpacesResource(appDB, config, auth.NewKeycloakResourceManager(config))
	workitemsResource := handler.NewWorkItemsResource(appDB, notificationChannel, config)
	httpEngine.GET("/api/v2/spaces", spacesResource.List)
	httpEngine.GET("/api/v2/spaces/:spaceID", spacesResource.GetByID)
	httpEngine.GET("/api/v2/spaces/:spaceID/workitems", workitemsResource.List)
	httpEngine.GET("/api/v2/workitems/:workitemID", workitemsResource.Show)
	// secured endpoints
	authGroup := httpEngine.Group("/")
	authGroup.Use(authMiddleware.MiddlewareFunc())
	// spaceAuthzService := authz.NewAuthzService(config, appDB)
	// authGroup.Use(authz.AuthzServiceManager(spaceAuthzService))
	authGroup.GET("/refresh_token", authMiddleware.RefreshHandler)
	authGroup.POST("/api/v2/spaces/", spacesResource.Create)
	authGroup.POST("/api/v2/spaces/:spaceID/workitems", workitemsResource.Create)
	authGroup.PATCH("/api/v2/workitems/:workitemID", authz.NewWorkItemEditorAuthorizator(appDB, config), workitemsResource.Update)
	return httpEngine
}
