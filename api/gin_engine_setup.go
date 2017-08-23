package api

import (
	"github.com/fabric8-services/fabric8-wit/api/handler"
	"github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/fabric8-services/fabric8-wit/notification"
	"github.com/fabric8-services/fabric8-wit/space/authz"
	"github.com/gin-gonic/gin"
)

// NewGinEngine instanciates a new HTTP engine to server the requests
func NewGinEngine(appDB *gormapplication.GormDB, notificationChannel notification.Channel, config *configuration.ConfigurationData) *gin.Engine {
	httpEngine := gin.Default()
	httpEngine.GET("/ping", func(c *gin.Context) {
		c.String(200, "pong")
	})
	authMiddleware := handler.NewJWTAuthMiddleware(appDB)
	spacesResource := handler.NewSpacesResource(appDB, config)
	workitemsResource := handler.NewWorkItemsResource(appDB, notificationChannel, config)
	httpEngine.GET("/api/spaces", spacesResource.List)
	httpEngine.GET("/api/spaces/:spaceID", spacesResource.GetByID)
	httpEngine.GET("/api/spaces/:spaceID/workitems", workitemsResource.List)
	httpEngine.GET("/api/workitems/:workitemID", workitemsResource.Show)
	// secured endpoints
	spaceAuthzService := authz.NewAuthzService(config, appDB)
	authGroup := httpEngine.Group("/")
	authGroup.Use(authMiddleware.MiddlewareFunc())
	authGroup.Use(authz.AuthzServiceHandler(spaceAuthzService))
	authGroup.GET("/refresh_token", authMiddleware.RefreshHandler)
	authGroup.POST("/api/spaces/:spaceID/workitems", workitemsResource.Create)
	authGroup.PATCH("/api/workitems/:workitemID", handler.WorkItemUpdateAuthorizator(appDB), workitemsResource.Update)
	return httpEngine
}
