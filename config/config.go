package config

import (
	"fmt"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
)

type AppConfig struct {
	Version    string        `yaml:"version"`
	Env        string        `yaml:"env"`
	Httpd      HttpdConfig   `yaml:"httpd"`
	RecordDir  string        `yaml:"recordDir"`
	LinuxUser  string        `yaml:"linuxUser"`
	PoolNum    int           `yaml:"poolNum"`
	Log        LogConfig     `yaml:"log"`
}

type HttpdConfig struct {
	Host string
	Port int
}

type LogConfig struct {
	Dir       string
	Name      string
	Format    string
	RetainDay int8
	Level     string
}

var (
	config *AppConfig
)

//通过viper初始化配置文件到结构体
func InitConfig() {
	dir,_ := os.Getwd()
	env := os.Getenv("ENV")
	if env == ""{
		env = "dev"
	}
	configPath := filepath.Join(dir,"config/"+env)
	// 设置读取的文件路径
	viper.AddConfigPath(configPath)
	// 设置读取的文件名
	viper.SetConfigName("config")
	// 设置文件的类型
	viper.SetConfigType("yaml")
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Sprintf("Read config error by %v \n",err))
	}
	var appConf *AppConfig
	if err :=viper.Unmarshal(&appConf); err !=nil{
		panic(fmt.Sprintf("Unmarshal config error by %v \n",err))
	}
	config = appConf
}

func GetConfig() *AppConfig{
	return config
}