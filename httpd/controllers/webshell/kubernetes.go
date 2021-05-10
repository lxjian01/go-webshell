package webshell

import (
	"github.com/gin-gonic/gin"
	"go-webshell/global/log"
	"go-webshell/httpd/middlewares"
	"go-webshell/httpd/services"
	"go-webshell/terminals/kubernetes"
)

func WsConnectKubernetes(c *gin.Context){
	// 获取参数
	projectCode := c.Param("project_code")
	moduleCode := c.Param("module_code")
	deployJobHostId := c.Param("deploy_job_host_id")
	host := c.Param("host")
	log.Infof("%s %s %d %s\n",projectCode,moduleCode,deployJobHostId,host)

	// 获取登陆用户信息
	loginUser := middlewares.GetLoginUser()
	// add login record
	loginId := services.InsertLoginRecord(loginUser.UserCode, projectCode, moduleCode,host,deployJobHostId)
	// new kubernetes terminal
	kubernetesTerminal, err := kubernetes.NewKubernetesTerminal(c.Writer, c.Request, nil)
	if err != nil {
		log.Errorf("New docker client error by %v \n", err)
		kubernetesTerminal.SendErrorMsg()
	}
	defer kubernetesTerminal.Close()
	// websocket close handler
	kubernetesTerminal.WsConn.SetCloseHandler(func(code int, text string) error {
		if kubernetesTerminal != nil{

			// add login out record
			services.UpdateLoginRecord(loginId)
		}
		return nil
	})
	// container exec
	//container := fmt.Sprintf("%s_%s",moduleCode,deployJobHostId)
	if err := kubernetes.Exec(kubernetesTerminal,"default","nginx-deployment-b5bd67766-cvwjw");err != nil{
		log.Error("Create container exec error by",err)
		kubernetesTerminal.SendErrorMsg()
	}
	err = kubernetesTerminal.CreateRecord(loginUser.UserCode, host)
	if err != nil{
		log.Error("Create record error by",err)
		kubernetesTerminal.SendErrorMsg()
	}
}
