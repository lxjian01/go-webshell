package webshell

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"go-webshell/httpd/middlewares"
	"go-webshell/httpd/services"
	"go-webshell/log"
	"go-webshell/pools"
	"go-webshell/terminals"
	"go-webshell/terminals/docker"
)

func WsConnectDocker(c *gin.Context){
	// 获取参数
	projectCode := c.Param("project_code")
	moduleCode := c.Param("module_code")
	deployJobHostId := c.Param("deploy_job_host_id")
	host := c.Param("host")
	log.Infof("%s %s %d %s\n",projectCode,moduleCode,deployJobHostId,host)
	// 初始化websocket
	tw, err := terminals.NewTerminalWebsocket(c.Writer, c.Request, nil)
	if err != nil {
		log.Error("Init websocket error by",err)
		return
	}
	log.Info("Websocket connect ok")
	defer tw.Ws.Close()

	// 获取登陆用户信息
	loginUser := middlewares.GetLoginUser()
	// add login record
	loginId := services.InsertLoginRecord(loginUser.UserCode, projectCode, moduleCode,host,deployJobHostId)
	// 定义client
	var dockerCli *docker.DockerClient
	// websocket close handler
	tw.Ws.SetCloseHandler(func(code int, text string) error {
		if dockerCli != nil{
			dockerCli.Close()
			// add login out record
			services.UpdateLoginRecord(loginId)
		}
		return nil
	})
	// new docker client
	dockerCli, err = docker.NewDockerClient(host)
	if err != nil {
		log.Errorf("New docker client error by %v \n", err)
		tw.SendErrorMsg()
	}
	// container exec
	container := fmt.Sprintf("%s_%s",moduleCode,deployJobHostId)
	if err := dockerCli.ContainerExecCreate(container);err != nil{
		log.Error("Create container exec error by",err)
		tw.SendErrorMsg()
	}
	record,err := terminals.CreateRecord(loginUser.UserCode, host)
	if err != nil{
		log.Error("Create record error by",err)
		tw.SendErrorMsg()
	}
	dockerCli.Record = record
	err = pools.Pool.Submit(func() {
		dockerCli.DockerReadWebsocketWrite(tw.Ws)
	})
	if err != nil{
		log.Error("Pool submit docker shell error by",err)
		tw.SendErrorMsg()
	}
	dockerCli.DockerWriteWebsocketRead(tw.Ws, loginUser.UserCode)
}