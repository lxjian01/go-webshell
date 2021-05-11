package webshell

import (
	"github.com/gin-gonic/gin"
	"go-webshell/global/log"
	"go-webshell/global/pools"
	"go-webshell/httpd/middlewares"
	"go-webshell/httpd/services"
	"go-webshell/terminals/linux"
)

func WsConnectLinux(c *gin.Context){
	// 获取参数
	projectCode := c.Param("project_code")
	moduleCode := c.Param("module_code")
	deployJobHostId := c.Param("deploy_job_host_id")
	host := c.Param("host")
	log.Infof("%s %s %d %s \n",projectCode,moduleCode,deployJobHostId,host)

	// 获取登陆用户信息
	loginUser := middlewares.GetLoginUser()
	// init linux client
	var linuxTerminal *linux.LinuxTerminal
	// add login record
	loginId := services.InsertLoginRecord(loginUser.UserCode, projectCode, moduleCode,host, deployJobHostId)
	// new linux terminal
	linuxTerminal, err := linux.NewLinuxTerminal(c.Writer, c.Request, nil, loginUser.UserCode, host)
	if err != nil{
		log.Error("Init ssh client error by ",err)
		linuxTerminal.SendErrorMsg()
	}
	defer linuxTerminal.Close()
	linuxTerminal.WsConn.SetCloseHandler(func(code int, text string) error {
		if linuxTerminal != nil{
			// add login out record
			services.UpdateLoginRecord(loginId)
		}
		return nil
	})
	if err := linuxTerminal.CreateSession();err != nil{
		log.Error("New ssh connect error by ",err)
		linuxTerminal.SendErrorMsg()
	}
	err = linuxTerminal.CreateRecordLinux(loginUser.UserCode, host)
	if err != nil{
		log.Error("Create record error by ",err)
		linuxTerminal.SendErrorMsg()
	}

	err = pools.Pool.Submit(func() {
		linuxTerminal.LinuxReadWebsocketWrite()
	})
	if err != nil{
		log.Error("Pool submit linux shell error by",err)
		linuxTerminal.SendErrorMsg()
	}
	linuxTerminal.LinuxWriteWebsocketRead()
}
