package api

import (
	"github.com/fabric8-services/fabric8-wit/api/resource"
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
	spacesResource := resource.NewSpacesResource(appDB)
	workitemsResource := resource.NewWorkItemsResource(appDB, config)
	httpEngine.GET("/api/spaces/:spaceID", spacesResource.GetByID)
	httpEngine.GET("/api/spaces/:spaceID/workitems", workitemsResource.List)
	httpEngine.GET("/api/workitems/:workitemID", workitemsResource.Show)
	return httpEngine
}
