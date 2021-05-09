package webshell

import (
	"github.com/gin-gonic/gin"
	"go-webshell/global/log"
	"go-webshell/global/pools"
	"go-webshell/httpd/middlewares"
	"go-webshell/httpd/services"
	"go-webshell/terminals"
	"go-webshell/terminals/linux"
)

func WsConnectLinux(c *gin.Context){
	// 获取参数
	projectCode := c.Param("project_code")
	moduleCode := c.Param("module_code")
	deployJobHostId := c.Param("deploy_job_host_id")
	host := c.Param("host")
	log.Infof("%s %s %d %s \n",projectCode,moduleCode,deployJobHostId,host)
	tw, err := terminals.NewTerminalWebsocket(c.Writer, c.Request, nil)
	if err != nil {
		log.Error("Init websocket error by ",err)
		return
	}
	log.Info("Websocket connect ok")
	defer tw.Ws.Close()

	// 获取登陆用户信息
	loginUser := middlewares.GetLoginUser()
	// init linux client
	var linuxCli *linux.LinuxClient
	var loginId int64
	// websocket close handler
	tw.Ws.SetCloseHandler(func(code int, text string) error {
		if linuxCli != nil{
			linuxCli.Close()
			services.UpdateLoginRecord(loginId)
		}
		return nil
	})
	linuxCli, err = linux.NewSshClient(host)
	if err != nil{
		log.Error("Init ssh client error by ",err)
		tw.SendErrorMsg()
	}
	if err := linuxCli.NewSession(100,100);err != nil{
		log.Error("New ssh connect error by ",err)
		tw.SendErrorMsg()
	}
	// add login record
	loginId = services.InsertLoginRecord(loginUser.UserCode, projectCode, moduleCode,host, deployJobHostId)
	record,err := terminals.CreateRecord(loginUser.UserCode, host)
	if err != nil{
		log.Error("Create record error by ",err)
		tw.SendErrorMsg()
	}
	linuxCli.Record = record

	err = pools.Pool.Submit(func() {
		linuxCli.LinuxReadWebsocketWrite(tw.Ws)
	})
	if err != nil{
		log.Error("Pool submit linux shell error by",err)
		tw.SendErrorMsg()
	}
	linuxCli.LinuxWriteWebsocketRead(tw.Ws, loginUser.UserCode)
}
