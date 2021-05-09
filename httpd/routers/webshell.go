package routers

import (
	"github.com/gin-gonic/gin"
	webshell2 "go-webshell/httpd/controllers/webshell"
)

func WebsocketRoutes(route *gin.Engine) {
	webshell := route.Group("/webshell")
	{
		webshell.GET("/docker/:project_code/:module_code/:host/:deploy_job_host_id/:token", webshell2.WsConnectDocker)

		webshell.GET("/linux/:project_code/:module_code/:host/:deploy_job_host_id/:token", webshell2.WsConnectLinux)
	}
}
