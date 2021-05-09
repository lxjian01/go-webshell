package controllers

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go-webshell/httpd/services"
	"go-webshell/log"
	"go-webshell/pools"
	"go-webshell/terminals"
	"go-webshell/terminals/linux"
	"strings"
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
		readLinuxToSendWebsocket(ws, linuxCli)
	})
	if err != nil{
		log.Error("Pool submit linux shell error by",err)
		wsSendErrorMsg(ws,"----error----")
	}
	var build strings.Builder
	for {
		// linux writer and websocket reader
		_, p, err := ws.ReadMessage()
		if err != nil {
			log.Error("Read websocket message error by ",err)
			return
		}
		cmd := string(p)
		if strings.HasPrefix(cmd, "{\"type\":\"resize\",\"rows\":"){
			resizeParams := new(resizeParams)
			if err := json.Unmarshal([]byte(cmd),&resizeParams);err != nil{
				log.Error("Unmarshal resize params error by ",err)
			}
			if err := linuxCli.SshConn.Session.WindowChange(resizeParams.Rows,resizeParams.Cols);err != nil{
				log.Error("Change ssh windows size error by ",err)
			}
		}else{
			writeCmdLog(&build,cmd,loginUser.UserCode,host,1)
			_,err1  := linuxCli.SshConn.StdinPipe.Write(p)
			if err1 != nil {
				log.Error("Websocket message copy to docker error by ",err)
				return
			}
		}
	}
}

func readLinuxToSendWebsocket(ws *websocket.Conn,linuxCli *linux.LinuxClient){
	for {
		// linux reader and websocket writer
		buf := make([]byte, 1024)
		n, err := linuxCli.SshConn.StdoutPipe.Read(buf)
		if err != nil {
			log.Error("Read docker message error by ",err)
			return
		}
		cmd := string(buf[:n])
		terminals.WriteRecord(linuxCli.Record,cmd)
		err1 := ws.WriteMessage(websocket.BinaryMessage, buf)
		if err1 != nil {
			log.Error("Docker message write to websocket error by ",err1)
			return
		}
	}
}
