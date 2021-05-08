package linux

import (
	"go-webshell/log"
	"go-webshell/terminals"
	"golang.org/x/crypto/ssh"
	"strings"
)

type LinuxServer struct {
	Host string
	Port int
}

func (c *LinuxServer) GetSshServerConfig() *ssh.ServerConfig{
	config := &ssh.ServerConfig{
		NoClientAuth: false,
		// Auth-related things should be constant-time to avoid timing attacks.
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			perm := &ssh.Permissions{Extensions: map[string]string{
				"pubkey": string(key.Marshal()),
			}}
			return perm, nil
		},
		KeyboardInteractiveCallback: func(conn ssh.ConnMetadata, challenge ssh.KeyboardInteractiveChallenge) (*ssh.Permissions, error) {
			return nil, nil
		},
		BannerCallback: func(conn ssh.ConnMetadata) string {
			var build strings.Builder
			build.WriteString("Welcome to Felix my friend ")
			build.WriteString(conn.User())
			build.WriteString("\n")
			build.WriteString("1 127.0.0.1 project \n")
			msg := build.String()
			log.Info(msg)
			return msg
		},
	}
	singer,err := terminals.GetSshSigner()
	if err != nil{
		log.Error("Get ssh singer error by",err)
	}
	config.AddHostKey(singer)
	return config
}

