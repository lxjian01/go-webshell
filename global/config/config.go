package config

import "go-webshell/config"

var conf *config.AppConfig

func SetAppConfig(c *config.AppConfig){
	conf = c
}

func GetAppConfig() *config.AppConfig {
	return conf
}