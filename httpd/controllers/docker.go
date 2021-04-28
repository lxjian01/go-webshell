package controllers

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go-webshell/log"
	"go-webshell/pools"
	"go-webshell/httpd/services"
	"go-webshell/terminals"
	"strings"
)

func WsConnectDocker(c *gin.Context){
	// 获取参数
	projectCode := c.Param("project_code")
	moduleCode := c.Param("module_code")
	deployJobHostId := c.Param("deploy_job_host_id")
	host := c.Param("host")
	log.Infof("%s %s %d %s\n",projectCode,moduleCode,deployJobHostId,host)
	// 初始化websocket
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Error("Init websocket error by",err)
		return
	}
	log.Info("Websocket connect ok")

	// 获取登陆用户信息
	loginUser := getLoginUser(c,ws)
	// 定义client
	var dockerCli *terminals.DockerClient
	var loginId int64
	// websocket close handler
	ws.SetCloseHandler(func(code int, text string) error {
		if !dockerCli.IsEmpty(){
			dockerCli.Close()
			// add login out record
			services.UpdateLoginRecord(loginId)
		}
		return nil
	})

	// init docker client
	container := fmt.Sprintf("%s_%s",moduleCode,deployJobHostId)
	dockerCli = &terminals.DockerClient{
		UserCode: loginUser.UserCode,
		Host: host,
		Container:container,
	}

	// add login record
	loginId = services.InsertLoginRecord(projectCode, moduleCode,host,deployJobHostId,loginUser.UserCode)

	if err := dockerCli.InitClient();err != nil{
		log.Error("Init client error by",err)
		wsSendErrorMsg(ws,"----error----")
		ws.Close()
	}
	if err := dockerCli.ContainerExecCreate();err != nil{
		log.Error("Create container exec error by",err)
		wsSendErrorMsg(ws,"----error----")
		ws.Close()
	}
	if err := dockerCli.ContainerExecAttach();err != nil{
		log.Error("Attach container exec error by",err)
		wsSendErrorMsg(ws,"----error----")
		ws.Close()
	}
	record,err := terminals.CreateRecord(host,loginUser.UserCode)
	if err != nil{
		log.Error("Create record error by",err)
		wsSendErrorMsg(ws,"----error----")
		ws.Close()
	}
	dockerCli.Record = record
	pools.Pool.Submit(func() {
		readDockerToSendWebsocket(ws,dockerCli)
	})
	var build strings.Builder
	for {
		// docker writer and websocket reader
		_, p, err := ws.ReadMessage()
		if err != nil {
			log.Error("Read websocket message error by",err)
			return
		}
		cmd := string(p)
		if strings.HasPrefix(cmd, "{\"type\":\"resize\",\"rows\":"){
			resizeParams := new(resizeParams)
			if err := json.Unmarshal([]byte(cmd),&resizeParams);err != nil{
				log.Error("Unmarshal resize params error by",err)
			}
			height := uint(resizeParams.Rows)
			width := uint(resizeParams.Cols)
			if err := dockerCli.ContainerExecResize(height,width);err != nil{
				log.Error("Change ssh windows size error by",err)
			}
		}else {
			writeCmdLog(&build, cmd, loginUser.UserCode, host, 0)
			_, err1 := dockerCli.Response.Conn.Write(p)
			if err1 != nil {
				log.Error("Websocket message copy to docker error by", err)
				return
			}
		}
	}
}

// read docker message to send websocket
func readDockerToSendWebsocket(ws *websocket.Conn,dockerCli *terminals.DockerClient){
	for {
		// docker reader and websocket writer
		buf := make([]byte, 10240)
		n, err := dockerCli.Response.Conn.Read(buf)
		if err != nil {
			log.Error("Read docker message error by",err)
			return
		}
		//cmd := strconv.Quote(string(buf[:n]))
		//a := strings.ReplaceAll(cmd, "[", "")
		//b := strings.ReplaceAll(a, "]", "")
		//fmt.Println(b)
		b := string(buf[:n])
		terminals.WriteRecord(dockerCli.Record,b)
		err = ws.WriteMessage(websocket.BinaryMessage, buf)
		if err != nil {
			log.Error("Docker message write to websocket error by",err)
			return
		}
	}
}
