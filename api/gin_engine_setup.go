package api

import (
	"github.com/fabric8-services/fabric8-wit/api/handler"
	"github.com/fabric8-services/fabric8-wit/configuration"
	"github.com/fabric8-services/fabric8-wit/gormapplication"
	"github.com/gin-gonic/gin"
)

// NewGinEngine instanciates a new HTTP engine to server the requests
func NewGinEngine(appDB *gormapplication.GormDB, config *configuration.ConfigurationData) *gin.Engine {
	httpEngine := gin.Default()
	httpEngine.GET("/ping", func(c *gin.Context) {
		c.String(200, "pong")
	})
	authMiddleware := handler.NewJWTAuthMiddleware(appDB)
	spacesResource := handler.NewSpacesResource(appDB, config)
	workitemsResource := handler.NewWorkItemsResource(appDB, config)
	httpEngine.GET("/api/spaces", spacesResource.List)
	httpEngine.GET("/api/spaces/:spaceID", spacesResource.GetByID)
	httpEngine.GET("/api/spaces/:spaceID/workitems", workitemsResource.List)
	httpEngine.GET("/api/workitems/:workitemID", workitemsResource.Show)
	// secured endpoints
	authGroup := httpEngine.Group("/")
	authGroup.Use(authMiddleware.MiddlewareFunc())
	authGroup.GET("/refresh_token", authMiddleware.RefreshHandler)
	authGroup.POST("/api/spaces/:spaceID/workitems", workitemsResource.Create)
	// authGroup.PATCH("/api/workitems/:workitemID", handler.NewWorkItemUpdateAuthorizator(appDB), workitemsResource.Update)
	return httpEngine
}
