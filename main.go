package main

import (
	"go-webshell/config"
	"go-webshell/httpd"
	"go-webshell/pools"
)

func main() {
	// init config
	config.InitConfig()
	// init ant pool
	pools.InitPool(config.GetConfig().PoolNum)
	// start http server
	httpConfig := config.GetConfig().Httpd
	httpd.StartHttpdServer(&httpConfig)
}
