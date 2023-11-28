package routers

import (
	"dvpn/controllers"
	"dvpn/middleware"
	"github.com/gin-gonic/gin"
)

type Router struct {
	Auth *middleware.AuthMiddleware

	HealthController  *controllers.HealthController
	DevicesController *controllers.DevicesController
	VPNController     *controllers.VPNController
}

func (r Router) RegisterRoutes(router gin.IRouter) {

	// Azure health check
	router.GET("/robots933456.txt", func(c *gin.Context) {
		middleware.RespondOK(c, nil)
	})

	//
	// Anonymous requests
	//
	router.GET("/health", r.HealthController.Status)
	router.GET("/versions", r.HealthController.GetSupportedAppVersions)
	router.POST("/device", r.DevicesController.CreateDevice)

	//
	// Authorized Requests
	//
	authorized := router.Group("/", r.Auth.RequireAuth)
	authorized.GET("/ip", r.VPNController.GetIPAddress)
	authorized.GET("/device", r.DevicesController.GetDevice)
	authorized.GET("/countries", r.VPNController.GetCountries)
	authorized.GET("/countries/:country_id/cities", r.VPNController.GetCities)
	authorized.GET("/countries/:country_id/cities/:city_id/servers", r.VPNController.GetServers)

	authorized.POST("/countries/:country_id/cities/:city_id/credentials", r.VPNController.ConnectToCity)
	authorized.POST("/countries/:country_id/cities/:city_id/credentials/:protocol", r.VPNController.ConnectToCity)
	authorized.POST("/countries/:country_id/cities/:city_id/servers/:server_id/credentials", r.VPNController.ConnectToServer)
}
