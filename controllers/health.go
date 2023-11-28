package controllers

import (
	"dvpn/middleware"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"os"
)

type HealthController struct {
	DB     *gorm.DB
	Logger *zap.SugaredLogger
}

func (h HealthController) Status(c *gin.Context) {
	err := h.DB.Raw(`SELECT 1`).Row().Err()
	if err != nil {
		h.Logger.Errorf("Error checking database health: %v", err)
		middleware.RespondErr(c, middleware.APIErrorUnknown, "Error checking database health")
		return
	}

	middleware.RespondOK(c, nil)
}

func (h HealthController) GetSupportedAppVersions(c *gin.Context) {
	middleware.RespondOK(c, gin.H{
		"API":     os.Getenv("MINIMAL_API_VERSION"),
		"ANDROID": os.Getenv("MINIMAL_ANDROID_VERSION"),
		"IOS":     os.Getenv("MINIMAL_IOS_VERSION"),
	})
}
