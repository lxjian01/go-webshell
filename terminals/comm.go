package terminals

import (
	"fmt"
	"github.com/spf13/viper"
	"go-webshell/log"
	"go-webshell/utils"
	"os"
	"path"
)

type Record struct {
	StartTime int
	File *os.File
}

func CreateRecord(userCode string, host string) (*Record,error){
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

func WriteRecord(record *Record, cmd string){
	t := float64(utils.DateUnixNano() - record.StartTime * 1e9) / 1e9
	cmdString := fmt.Sprintf("[%.6f,\"%s\",%s]\n",t,"o",cmd)
	log.Info(cmdString)
	_,err := record.File.WriteString(cmdString)
	if err != nil{
		log.Errorf("Write cmd % in file error by %v \n",cmd,err)
	}
}
