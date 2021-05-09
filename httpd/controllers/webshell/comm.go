package webshell

import (
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go-webshell/httpd/middlewares"
	"go-webshell/log"
	"net/http"
	"reflect"
)

var upgrader = websocket.Upgrader{
	HandshakeTimeout:6*60,
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
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