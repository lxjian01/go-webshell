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

type Terminal struct {
	Ws *websocket.Conn
	record *Record
}

func NewTerminal(w http.ResponseWriter, r *http.Request, responseHeader http.Header) (*Terminal, error) {
	var terminal Terminal
	var upgrader = websocket.Upgrader{
		HandshakeTimeout:6*60,
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	ws, err := upgrader.Upgrade(w, r, responseHeader)
	terminal.Ws = ws
	return &terminal, err
}

// websocket send message
func (t *Terminal) SendMsg(msg string)  {
	if err := t.Ws.WriteMessage(websocket.BinaryMessage,[]byte(msg));err != nil{
		log.Errorf("Websocket write message %s error by %v \n",err)
	}else{
		log.Infof("Websocket write message %s ok by %v \n",err)
	}
}

// websocket send message
func (t *Terminal) SendErrorMsg()  {
	t.SendMsg("----error----")
}

func (t *Terminal) CreateRecord(userCode string, host string) error {
	recordDir := globalConf.GetAppConfig().RecordDir
	if !utils.IsExist(recordDir){
		_, err := utils.CreateDir(recordDir)
		if err != nil {
			return err
		}
	}

	time := utils.DateUnix()
	filename := fmt.Sprintf("docker_%s_%s_%d.cast",host,userCode,time)
	file := path.Join(recordDir, filename)
	f, err := os.Create(file) //创建文件
	if err != nil{
		return err
	}
	record := &Record{
		StartTime: time,
		File: f,
	}
	recordStart := fmt.Sprintf("{\"version\": 2, \"width\": 237, \"height\": 55, \"timestamp\": %d, \"env\": {\"SHELL\": \"/bin/bash\", \"TERM\": \"linux\"}}\n",record.StartTime)
	_, errw := record.File.WriteString(recordStart)
	t.record = record
	return errw
}

func (t *Terminal) WriteRecord(cmd string){
	timeMinus := float64(utils.DateUnixNano() - t.record.StartTime * 1e9) / 1e9
	cmdString := fmt.Sprintf("[%.6f,\"%s\",%s]\n",timeMinus,"o",cmd)
	log.Info(cmdString)
	_,err := t.record.File.WriteString(cmdString)
	if err != nil{
		log.Errorf("Write cmd % in file error by %v \n",cmd,err)
	}
}

func (t *Terminal) Close(){
	// close websocket
	if t.Ws != nil {
		if err := t.Ws.Close(); err != nil{
			log.Errorf("Start close websocket error by %s \n", err.Error())
		}else{
			log.Info("Close docker websocket ok")
		}
	}
	// close record file
	if t.record != nil {
		if err := t.record.File.Close(); err != nil{
			log.Errorf("Start close terminal record error by %s \n", err.Error())
		}else{
			log.Info("Close terminal record ok")
		}

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
