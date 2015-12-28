package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ovh/tatsamon/internal"
)

// SystemController contains all methods about version
type SystemController struct{}

//GetVersion returns version of tat
func (*SystemController) GetVersion(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{"version": internal.VERSION})
}
