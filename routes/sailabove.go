package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/ovh/tatsamon/controllers"
)

// InitRoutesApplications initialized routes for Sailabove Controller /application
func InitRoutesApplications(router *gin.Engine) {
	c := &controllers.SailaboveController{}
	router.GET("/applications", c.ListApplications)
}

// InitRoutesCheck initialized routes for Sailabove Controller /containers/check
func InitRoutesCheck(router *gin.Engine) {
	c := &controllers.SailaboveController{}
	router.GET("/containers/check", c.CheckApplications)
}
