package terminals

import (
	"fmt"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"go-webshell/log"
	"go-webshell/utils"
	"os"
	"path"
	"path/filepath"
)

type Record struct {
	StartTime int
	File *os.File
}

func CreateRecord(host string,userCode string) (*Record,error){
	recordDir := viper.GetString("RecordDir")
	if !utils.IsExist(recordDir){
		_, err := utils.CreateDir(recordDir)
		if err != nil {
			return nil,err
		}
	}

	time := utils.DateUnix()
	filename := fmt.Sprintf("docker_%s_%s_%d.cast",host,userCode,time)
	file := path.Join(recordDir, filename)
	f, err := os.Create(file) //创建文件
	if err != nil{
		return nil,err
	}
	record := &Record{
		StartTime: time,
		File: f,
	}
	t := fmt.Sprintf("{\"version\": 2, \"width\": 237, \"height\": 55, \"timestamp\": %d, \"env\": {\"SHELL\": \"/bin/bash\", \"TERM\": \"linux\"}}\n",record.StartTime)
	_,errw :=record.File.WriteString(t)
	return record,errw
}

func WriteRecord(record *Record,cmd string){
	t := float64(utils.DateUnixNano() - record.StartTime * 1e9) / 1e9
	cmdString := fmt.Sprintf("[%.6f,\"%s\",%s]\n",t,"o",cmd)
	log.Info(cmdString)
	_,err := record.File.WriteString(cmdString)
	if err != nil{
		log.Errorf("Write cmd % in file error by %v \n",cmd,err)
	}
}

func getSshSigner() (ssh.Signer,error) {
	dir,_ := os.Getwd()
	log.Info("Linux path is",dir)
	env := viper.GetString("Env")
	cafile := filepath.Join(dir,"/config/",env,"/keys/linux/id_rsa")
	key, err := ioutil.ReadFile(cafile)
	if err != nil {
		log.Error("ssh key file read failed", err)
	}
	// Create the Signer for this private key.
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		log.Error("ssh key signer failed", err)
	}
	return signer, err
}
