package webshell

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"go-webshell/global/log"
	"go-webshell/global/pools"
	"go-webshell/httpd/middlewares"
	"go-webshell/httpd/services"
	"go-webshell/terminals/docker"
)

func WsConnectDocker(c *gin.Context){
	// 获取参数
	projectCode := c.Param("project_code")
	moduleCode := c.Param("module_code")
	deployJobHostId := c.Param("deploy_job_host_id")
	host := c.Param("host")
	log.Infof("%s %s %d %s\n",projectCode,moduleCode,deployJobHostId,host)

	// 获取登陆用户信息
	loginUser := middlewares.GetLoginUser()
	// add login record
	loginId := services.InsertLoginRecord(loginUser.UserCode, projectCode, moduleCode,host,deployJobHostId)
	// new docker terminal
	dockerTerminal, err := docker.NewDockerTerminal(c.Writer, c.Request, nil, loginUser.UserCode, host)
	if err != nil {
		log.Errorf("New docker client error by %v \n", err)
		dockerTerminal.SendErrorMsg()
	}
	defer dockerTerminal.Close()
	// websocket close handler
	dockerTerminal.WsConn.SetCloseHandler(func(code int, text string) error {
		if dockerTerminal != nil{
			// add login out record
			services.UpdateLoginRecord(loginId)
		}
		return nil
	})
	// container exec
	container := fmt.Sprintf("%s_%s",moduleCode,deployJobHostId)
	if err := dockerTerminal.ContainerExecCreate(container);err != nil{
		log.Error("Create container exec error by",err)
		dockerTerminal.SendErrorMsg()
	}
	err = dockerTerminal.CreateRecord(loginUser.UserCode, host)
	if err != nil{
		log.Error("Create record error by",err)
		dockerTerminal.SendErrorMsg()
	}
	err = pools.Pool.Submit(func() {
		dockerTerminal.DockerReadWebsocketWrite()
	})
	if err != nil{
		log.Error("Pool submit docker shell error by",err)
		dockerTerminal.SendErrorMsg()
	}
	dockerTerminal.DockerWriteWebsocketRead()
}