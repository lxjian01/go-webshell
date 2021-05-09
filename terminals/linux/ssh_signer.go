package linux

import (
	globalConf "go-webshell/global/config"
	"go-webshell/global/log"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"os"
	"path/filepath"
)

func GetSshSigner() (ssh.Signer, error) {
	dir,_ := os.Getwd()
	log.Info("Linux path is",dir)
	env := globalConf.GetAppConfig().Env
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
