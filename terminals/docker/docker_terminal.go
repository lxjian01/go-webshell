package docker

import "C"
import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/tlsconfig"
	"github.com/gorilla/websocket"
	globalConf "go-webshell/global/config"
	"go-webshell/global/log"
	"go-webshell/terminals"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var (
	version = "1.38"
	ctx = context.Background()
	)

type DockerTerminal struct{
	terminals.BaseTerminal
	host string
	cli *client.Client
	execId string
	Response types.HijackedResponse
}

func NewDockerTerminal(w http.ResponseWriter, r *http.Request, responseHeader http.Header, host string) (*DockerTerminal, error) {

	// 初始化websocket
	wsConn, err := terminals.NewWebsocket(w, r, responseHeader)
	if err != nil {
		log.Error("Init websocket error by",err)
		return nil, err
	}
	log.Info("Websocket connect ok")

	var c DockerTerminal
	c.host = host
	c.WsConn = wsConn
	options := getOptions()
	tlsConfig, err := tlsconfig.Client(options)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}
	hostCon := fmt.Sprintf("tcp://%s:2375", c.host)
	cli, err1 := client.NewClient(hostCon, version,httpClient,nil)
	c.cli = cli
	return &c, err1
}

func getOptions() tlsconfig.Options{
	dir, _ := os.Getwd()
	log.Info("Docker path is",dir)
	env := globalConf.GetAppConfig().Env
	caFile := filepath.Join(dir,"/config/",env,"/keys/docker/ca.pem")
	certFile :=  filepath.Join(dir,"/config/",env,"/keys/docker/client-cert.pem")
	keyFile :=  filepath.Join(dir,"/config/",env,"/keys/docker/client-key.pem")
	options := tlsconfig.Options{
		CAFile:            caFile,
		CertFile:          certFile,
		KeyFile:           keyFile,
		InsecureSkipVerify: true,
	}
	return options
}

func (terminal *DockerTerminal) ContainerExecCreate(container string) error{
	cmd := []string{
		"/bin/sh",
		"-c",
		"TERM=xterm-256color; export TERM; /bin/bash"}
	//envs := []string{
	//	"LINES=$(tput lines)",
	//	"COLUMNS=$(tput cols)",
	//}
	execCreateConf := types.ExecConfig{
		AttachStderr: true,
		AttachStdin:  true,
		AttachStdout: true,
		//Env: envs,
		Cmd: cmd,
		Tty:          true,
		Detach:       true,
	}
	exec,err := terminal.cli.ContainerExecCreate(ctx, container, execCreateConf)
	if err != nil {
		return err
	}
	terminal.execId = exec.ID

	execAttachConf := types.ExecStartCheck{
		Detach: false,
		Tty: true,
	}
	conn,err1 := terminal.cli.ContainerExecAttach(ctx,terminal.execId,execAttachConf)
	if err1 != nil {
		return err1
	}
	terminal.Response = conn
	return nil
}

func (terminal *DockerTerminal) ContainerExecResize(height uint, width uint) error{
	options := types.ResizeOptions{
		Height:height,
		Width: width,
	}
	err := terminal.cli.ContainerExecResize(ctx, terminal.execId, options)
	return err
}

// read docker message to send websocket
func (terminal *DockerTerminal) DockerReadWebsocketWrite(){
	for {
		// docker reader and websocket writer
		buf := make([]byte, 10240)
		n, err := terminal.Response.Conn.Read(buf)
		if err != nil {
			log.Error("Read docker message error by",err)
			return
		}
		//cmd := strconv.Quote(string(buf[:n]))
		//a := strings.ReplaceAll(cmd, "[", "")
		//b := strings.ReplaceAll(a, "]", "")
		//fmt.Println(b)
		b := string(buf[:n])
		terminal.WriteRecord(b)
		err = terminal.WsConn.WriteMessage(websocket.BinaryMessage, buf)
		if err != nil {
			log.Error("Docker message write to websocket error by",err)
			return
		}
	}
}

func (terminal *DockerTerminal) DockerWriteWebsocketRead(userCode string){
	var build strings.Builder
	for {
		// docker writer and websocket reader
		_, p, err := terminal.WsConn.ReadMessage()
		if err != nil {
			log.Error("Read websocket message error by",err)
			return
		}
		cmd := string(p)
		if strings.HasPrefix(cmd, "{\"type\":\"resize\",\"rows\":"){
			var resizeParams terminals.ResizeParams
			if err := json.Unmarshal([]byte(cmd),&resizeParams);err != nil{
				log.Error("Unmarshal resize params error by",err)
			}
			height := uint(resizeParams.Rows)
			width := uint(resizeParams.Cols)
			if err := terminal.ContainerExecResize(height,width);err != nil{
				log.Error("Change ssh windows size error by",err)
			}
		}else {
			terminals.WriteCmdLog(&build, cmd, userCode, terminal.host, 0)
			_, err1 := terminal.Response.Conn.Write(p)
			if err1 != nil {
				log.Error("Websocket message copy to docker error by", err)
				return
			}
		}
	}
}

func (terminal *DockerTerminal) Close() {
	// close docker client
	if terminal.cli != nil {
		if err := terminal.cli.Close();err != nil {
			log.Errorf("Close docker client exec id is %s error by % \n", terminal.execId, err.Error())
		}else{
			log.Infof("Close docker client ok by exec id is %s \n", terminal.execId)
		}

		terminal.Response.Close()
		log.Infof("End close docker client response by exec id is %s ok \n", terminal.execId)
	}
	terminal.CloseWs()
	terminal.CloseRecordFile()
}