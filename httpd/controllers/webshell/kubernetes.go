package webshell

import (
	"github.com/gin-gonic/gin"
	"go-webshell/global/log"
	"go-webshell/httpd/middlewares"
	"go-webshell/httpd/services"
	"go-webshell/terminals"
	"go-webshell/terminals/kubernetes"
)

func WsConnectKubernetes(c *gin.Context){
	// 获取参数
	projectCode := c.Param("project_code")
	moduleCode := c.Param("module_code")
	deployJobHostId := c.Param("deploy_job_host_id")
	host := c.Param("host")
	log.Infof("%s %s %d %s\n",projectCode,moduleCode,deployJobHostId,host)
	// 初始化websocket
	terminal, err := terminals.NewTerminal(c.Writer, c.Request, nil)
	if err != nil {
		log.Error("Init websocket error by",err)
		return
	}
	log.Info("Websocket connect ok")
	defer terminal.Close()

	// 获取登陆用户信息
	loginUser := middlewares.GetLoginUser()
	// add login record
	loginId := services.InsertLoginRecord(loginUser.UserCode, projectCode, moduleCode,host,deployJobHostId)
	// 定义client
	var dockerCli *kubernetes.TerminalSession
	// websocket close handler
	terminal.Ws.SetCloseHandler(func(code int, text string) error {
		if dockerCli != nil{
			dockerCli.Close()
			// add login out record
			services.UpdateLoginRecord(loginId)
		}
		return nil
	})
	// new docker client
	dockerCli, err = kubernetes.NewTerminalSession(terminal.Ws)
	if err != nil {
		log.Errorf("New docker client error by %v \n", err)
		terminal.SendErrorMsg()
	}
	// container exec
	//container := fmt.Sprintf("%s_%s",moduleCode,deployJobHostId)
	if err := kubernetes.Exec(dockerCli,"default","nginx-deployment-b5bd67766-cvwjw");err != nil{
		log.Error("Create container exec error by",err)
		terminal.SendErrorMsg()
	}
	err = terminal.CreateRecord(loginUser.UserCode, host)
	if err != nil{
		log.Error("Create record error by",err)
		terminal.SendErrorMsg()
	}
}