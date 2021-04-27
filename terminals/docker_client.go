package terminals

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/tlsconfig"
	"github.com/spf13/viper"
	"go-webshell/log"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
)

var (
	version = "1.38"
	ctx = context.Background()
	)

type DockerClient struct{
	UserCode string
	Host  string
	Container string
	cli *client.Client
	execId string
	Response types.HijackedResponse
	Record *Record
}



func (c DockerClient) IsEmpty() bool {
	return reflect.DeepEqual(c, DockerClient{})
}

func (c *DockerClient) InitClient() error {
	options := c.getOptions()
	tlsc, err := tlsconfig.Client(options)
	if err != nil {
		fmt.Println(err)
		return err
	}
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsc,
		},
	}
	hostCon := fmt.Sprintf("tcp://%s:2375",c.Host)
	cli, err1 := client.NewClient(hostCon, version,httpClient,nil)
	if err1 != nil{
		return err1
	}
	c.cli = cli
	return nil
}

func (c *DockerClient) getOptions() tlsconfig.Options{
	dir,_ := os.Getwd()
	log.Info("Docker path is",dir)
	env := viper.GetString("Env")
	caFile := filepath.Join(dir,"/config/",env,"/keys/docker/ca.pem")
	certfile :=  filepath.Join(dir,"/config/",env,"/keys/docker/cert.pem")
	keyfile :=  filepath.Join(dir,"/config/",env,"/keys/docker/key.pem")
	options := tlsconfig.Options{
		CAFile:            caFile,
		CertFile:          certfile,
		KeyFile:           keyfile,
		InsecureSkipVerify: true,
	}
	return options
}

func (c *DockerClient) ContainerExecCreate() error{
	cmds := []string{
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
		Cmd: cmds,
		Tty:          true,
		Detach:       true,
	}
	exec,err := c.cli.ContainerExecCreate(ctx,c.Container,execCreateConf)
	c.execId = exec.ID
	return err
}

func (c *DockerClient) ContainerExecAttach() error{
	execAttachConf := types.ExecConfig{
		Tty: true,
		Detach: false,
	}
	conn,err := c.cli.ContainerExecAttach(ctx,c.execId,execAttachConf)
	c.Response = conn
	return err
}

func (c *DockerClient) ContainerExecResize(height uint,width uint) error{
	options := types.ResizeOptions{
		Height:height,
		Width: width,
	}
	err := c.cli.ContainerExecResize(ctx,c.execId,options)
	return err
}

func (c *DockerClient) Close() {
	c.Record.File.Close()
	log.Info("Start close docker client conn by exec id is",c.execId)
	c.Response.Close()
	log.Info("End close docker client conn by exec id is",c.execId)
	if err := c.cli.Close();err != nil {
		log.Error("Close docker client error by exec id is",c.execId)
	}else{
		log.Info("Close docker client ok by exec id is",c.execId)
	}
}