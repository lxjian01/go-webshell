package terminals

import (
	"fmt"
	"github.com/gorilla/websocket"
	globalConf "go-webshell/global/config"
	"go-webshell/global/log"
	"go-webshell/httpd/services"
	"go-webshell/utils"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

type Record struct {
	StartTime int
	File *os.File
}

type ResizeParams struct {
	Ptype   string   `json:"type"`
	Rows    int      `json:"rows"`
	Cols    int      `json:"cols"`
	Height  int      `json:"height"`
	Width   int      `json:"width"`
}

type BaseTerminal struct {
	WsConn *websocket.Conn
	record *Record
}

func NewWebsocket(w http.ResponseWriter, r *http.Request, responseHeader http.Header) (*websocket.Conn, error) {
	var upgrader = websocket.Upgrader{
		HandshakeTimeout: time.Second * 6,
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	ws, err := upgrader.Upgrade(w, r, responseHeader)
	return ws, err
}

// websocket send message
func (t *BaseTerminal) SendMsg(msg string)  {
	if err := t.WsConn.WriteMessage(websocket.BinaryMessage,[]byte(msg));err != nil{
		log.Errorf("Websocket write message %s error by %v \n",err)
	}else{
		log.Infof("Websocket write message %s ok by %v \n",err)
	}
}

// websocket send message
func (t *BaseTerminal) SendErrorMsg()  {
	t.SendMsg("----error----")
}

func (t *BaseTerminal) createRecord(name string, userCode string, host string) error {
	recordDir := globalConf.GetAppConfig().RecordDir
	if !utils.IsExist(recordDir){
		_, err := utils.CreateDir(recordDir)
		if err != nil {
			return err
		}
	}

	nowTime := utils.DateUnix()
	filename := fmt.Sprintf("%s_%s_%s_%d.cast", name, host, userCode, nowTime)
	file := path.Join(recordDir, filename)
	f, err := os.Create(file) //创建文件
	if err != nil{
		return err
	}
	record := &Record{
		StartTime: nowTime,
		File: f,
	}
	recordStart := fmt.Sprintf("{\"version\": 2, \"width\": 237, \"height\": 55, \"timestamp\": %d, \"env\": {\"SHELL\": \"/bin/bash\", \"TERM\": \"linux\"}}\n",record.StartTime)
	_, err = record.File.WriteString(recordStart)
	t.record = record
	return err
}

func (t *BaseTerminal) CreateRecordLinux(userCode string, host string) error {
	return t.createRecord("linux", userCode, host)
}

func (t *BaseTerminal) CreateRecordDocker(userCode string, host string) error {
	return t.createRecord("docker", userCode, host)
}

func (t *BaseTerminal) CreateRecordKubernetes(userCode string, host string) error {
	return t.createRecord("kubernetes", userCode, host)
}

func (t *BaseTerminal) WriteRecord(cmd string){
	timeMinus := float64(utils.DateUnixNano() - t.record.StartTime * 1e9) / 1e9
	cmdString := fmt.Sprintf("[%.6f,\"%s\",%s]\n",timeMinus,"o",cmd)
	log.Info(cmdString)
	_,err := t.record.File.WriteString(cmd)
	if err != nil{
		log.Errorf("Write cmd % in file error by %v \n",cmd,err)
	}
}

// close websocket
func (t *BaseTerminal) CloseWs(){
	if t.WsConn != nil {
		if err := t.WsConn.Close(); err != nil{
			log.Errorf("Start close websocket error by %s \n", err.Error())
		}else{
			log.Info("Close docker websocket ok")
		}
	}
}

// close record file
func (t *BaseTerminal) CloseRecordFile(){
	if t.record != nil {
		if err := t.record.File.Close(); err != nil{
			log.Errorf("Start close terminal record error by %s \n", err.Error())
		}else{
			log.Info("Close terminal record ok")
		}

	}
}

// write oper commands
func (t *BaseTerminal) WriteCmdLog(build *strings.Builder, msg string, userCode string, host string, mtype int)  {
	if msg == "\r"{
		cmd := build.String()
		if mtype == 0 {
			services.AddDockerOperRecord(cmd, userCode, host)
		} else if mtype == 1 {
			services.AddLinuxOperRecord(cmd, userCode, host)
		}else{
			services.AddKubernetesOperRecord(cmd, userCode, host)
		}
		build.Reset()
	}else{
		build.WriteString(msg)
	}
}
