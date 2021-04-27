package main

import (
	"go-webshell/config"
	"go-webshell/httpd"
)

func main() {
	// init config
	config.InitConfig()
	// start http server
	httpConfig := config.GetConfig().Httpd
	httpd.StartHttpdServer(&httpConfig)
}
