package docker

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/tlsconfig"
	"github.com/spf13/viper"
	"go-webshell/log"
	"go-webshell/terminals"
	"net/http"
	"os"
	"path/filepath"
)

var (
	version = "1.38"
	ctx = context.Background()
	)

type DockerClient struct{
	cli *client.Client
	execId string
	Response types.HijackedResponse
	Record *terminals.Record
}

func NewDockerClient(host string) (*DockerClient, error) {
	var c DockerClient
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
	hostCon := fmt.Sprintf("tcp://%s:2375", host)
	cli, err1 := client.NewClient(hostCon, version,httpClient,nil)
	c.cli = cli
	return &c, err1
}

func getOptions() tlsconfig.Options{
	dir, _ := os.Getwd()
	log.Info("Docker path is",dir)
	env := viper.GetString("Env")
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

func (c *DockerClient) ContainerExecCreate(container string) error{
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
	exec,err := c.cli.ContainerExecCreate(ctx, container, execCreateConf)
	if err != nil {
		return err
	}
	c.execId = exec.ID

	execAttachConf := types.ExecStartCheck{
		Detach: false,
		Tty: true,
	}
	conn,err1 := c.cli.ContainerExecAttach(ctx,c.execId,execAttachConf)
	if err1 != nil {
		return err1
	}
	c.Response = conn
	return nil
}

func (c *DockerClient) ContainerExecResize(height uint, width uint) error{
	options := types.ResizeOptions{
		Height:height,
		Width: width,
	}
	err := c.cli.ContainerExecResize(ctx,c.execId,options)
	return err
}

func (c *DockerClient) Close() {
	// close docker client
	if c.cli != nil {
		if err := c.cli.Close();err != nil {
			log.Errorf("Close docker client exec id is %s error by % \n", c.execId, err.Error())
		}else{
			log.Infof("Close docker client ok by exec id is %s \n", c.execId)
		}
		// close record file
		if c.Record != nil {
			if err := c.Record.File.Close(); err != nil{
				log.Errorf("Start close docker client record exec id %s error by %s \n", c.execId, err.Error())
			}else{
				log.Infof("Close docker client record exec id %s ok", c.execId)
			}
			c.Response.Close()
			log.Infof("End close docker client response by exec id is %s ok \n", c.execId)
		}
	}
}