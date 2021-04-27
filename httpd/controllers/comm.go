package controllers

import (
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go-webshell/log"
	"go-webshell/httpd/middlewares"
	"go-webshell/httpd/services"
	"net/http"
	"reflect"
	"strings"
)

var upgrader = websocket.Upgrader{
	HandshakeTimeout:6*60,
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type resizeParams struct {
	Ptype   string   `json:"type"`
	Rows    int      `json:"rows"`
	Cols    int      `json:"cols"`
	Height  int      `json:"height"`
	Width   int      `json:"width"`
}

// write oper commands
func writeCmdLog(build *strings.Builder,msg string,userCode string,host string,mtype int)  {
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

// websocket send message
func wsSendErrorMsg(ws *websocket.Conn,msg string)  {
	if err := ws.WriteMessage(websocket.BinaryMessage,[]byte(msg));err != nil{
		log.Error("Websocket write error message error by",err)
	}else{
		log.Info("Websocket write error message ok by",err)
	}
}

// 获取登陆用户
func getLoginUser(c *gin.Context,ws *websocket.Conn) *middlewares.LoginUser {
	user := c.MustGet("loginUser")
	pointer := reflect.ValueOf(user)
	loginUser := pointer.Interface().(*middlewares.LoginUser)
	if loginUser == nil {
		wsSendErrorMsg(ws,"----no login----")
		ws.Close()
	}
	return loginUser
}