package webshell

import (
	"github.com/gin-gonic/gin"
	"go-webshell/httpd/services"
	"go-webshell/log"
	"go-webshell/pools"
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
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Error("Init websocket error by ",err)
		return
	}
	log.Info("Websocket connect ok")
	defer ws.Close()
	// 获取登陆用户信息
	loginUser := getLoginUser(c,ws)
	// init linux client
	var linuxCli *linux.LinuxClient
	var loginId int64
	// websocket close handler
	ws.SetCloseHandler(func(code int, text string) error {
		if linuxCli != nil{
			linuxCli.Close()
			services.UpdateLoginRecord(loginId)
		}
		return nil
	})
	linuxCli, err = linux.NewSshClient(host)
	if err != nil{
		log.Error("Init ssh client error by ",err)
		wsSendErrorMsg(ws,"----error----")
	}
	if err := linuxCli.NewSession(100,100);err != nil{
		log.Error("New ssh connect error by ",err)
		wsSendErrorMsg(ws,"----error----")
	}
	// add login record
	loginId = services.InsertLoginRecord(loginUser.UserCode, projectCode, moduleCode,host, deployJobHostId)
	record,err := terminals.CreateRecord(loginUser.UserCode, host)
	if err != nil{
		log.Error("Create record error by ",err)
		wsSendErrorMsg(ws,"----error----")
	}
	linuxCli.Record = record

	err = pools.Pool.Submit(func() {
		linuxCli.LinuxReadWebsocketWrite(ws)
	})
	if err != nil{
		log.Error("Pool submit linux shell error by",err)
		wsSendErrorMsg(ws,"----error----")
	}
	linuxCli.LinuxWriteWebsocketRead(ws, loginUser.UserCode)
}
