package terminals

import (
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/spf13/viper"
	"go-webshell/httpd/services"
	"go-webshell/log"
	"go-webshell/utils"
	"net/http"
	"os"
	"path"
	"strings"
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

type TerminalWebsocket struct {
	Ws *websocket.Conn
}

func NewTerminalWebsocket(w http.ResponseWriter, r *http.Request, responseHeader http.Header) (*TerminalWebsocket, error) {
	var tw TerminalWebsocket
	var upgrader = websocket.Upgrader{
		HandshakeTimeout:6*60,
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	ws, err := upgrader.Upgrade(w, r, responseHeader)
	tw.Ws = ws
	return &tw, err
}

// websocket send message
func (tw *TerminalWebsocket) SendMsg(msg string)  {
	if err := tw.Ws.WriteMessage(websocket.BinaryMessage,[]byte(msg));err != nil{
		log.Errorf("Websocket write message %s error by %v \n",err)
	}else{
		log.Infof("Websocket write message %s ok by %v \n",err)
	}
}

// websocket send message
func (tw *TerminalWebsocket) SendErrorMsg()  {
	tw.SendMsg("----error----")
}

func CreateRecord(userCode string, host string) (*Record,error){
	recordDir := viper.GetString("RecordDir")
	if !utils.IsExist(recordDir){
		_, err := utils.CreateDir(recordDir)
		if err != nil {
			return nil,err
		}
	}

	time := utils.DateUnix()
	filename := fmt.Sprintf("docker_%s_%s_%d.cast",host,userCode,time)
	file := path.Join(recordDir, filename)
	f, err := os.Create(file) //创建文件
	if err != nil{
		return nil,err
	}
	record := &Record{
		StartTime: time,
		File: f,
	}
	t := fmt.Sprintf("{\"version\": 2, \"width\": 237, \"height\": 55, \"timestamp\": %d, \"env\": {\"SHELL\": \"/bin/bash\", \"TERM\": \"linux\"}}\n",record.StartTime)
	_,errw :=record.File.WriteString(t)
	return record,errw
}

func WriteRecord(record *Record, cmd string){
	t := float64(utils.DateUnixNano() - record.StartTime * 1e9) / 1e9
	cmdString := fmt.Sprintf("[%.6f,\"%s\",%s]\n",t,"o",cmd)
	log.Info(cmdString)
	_,err := record.File.WriteString(cmdString)
	if err != nil{
		log.Errorf("Write cmd % in file error by %v \n",cmd,err)
	}
}

// write oper commands
func WriteCmdLog(build *strings.Builder,msg string,userCode string,host string,mtype int)  {
	if msg == "\r"{
		cmd := build.String()
		if mtype == 0{
			services.AddDockerOperRecord(cmd,userCode,host)
		}else{
			services.AddLinuxOperRecord(cmd,userCode,host)
		}
		build.Reset()
	}else{
		build.WriteString(msg)
	}
}
