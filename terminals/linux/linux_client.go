package linux

import (
	"bytes"
	"github.com/spf13/viper"
	"go-webshell/log"
	"go-webshell/terminals"
	"golang.org/x/crypto/ssh"
	"io"
	"net"
	"strconv"
	"sync"
	"time"
)

// write data to WebSocket
// the data comes from ssh server.
type wsBufferWriter struct {
	buffer bytes.Buffer
	mu     sync.Mutex
}

// implement Write interface to write bytes from ssh server into bytes.Buffer.
func (w *wsBufferWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buffer.Write(p)
}

type LinuxClient struct {
	Host string
	Cli *ssh.Client
	SshConn *SshConn
	Record *terminals.Record
	IsClose bool
}

// connect to ssh server using ssh session.
type SshConn struct {
	// calling Write() to write data into ssh server
	StdinPipe io.WriteCloser
	StdoutPipe io.Reader
	// Write() be called to receive data from ssh server
	// ComboOutput *wsBufferWriter
	Session     *ssh.Session
}

func (c *LinuxClient) publicKeyAuthFunc(singer ssh.Signer) ssh.AuthMethod{
	return ssh.PublicKeys(singer)
}

func (c *LinuxClient) InitSshClient() error {
	LinuxUser := viper.GetString("LinuxUser")
	config := &ssh.ClientConfig{
		Timeout:         time.Second * 5,
		User:            LinuxUser,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //这个可以， 但是不够安全
		//HostKeyCallback: hostKeyCallBackFunc(h.Host),
	}
	singer,err := terminals.GetSshSigner()
	if err != nil{
		return err
	}
	config.Auth = []ssh.AuthMethod{c.publicKeyAuthFunc(singer)}
	addr := net.JoinHostPort(c.Host, strconv.Itoa(22))
	cli, err := ssh.Dial("tcp", addr, config)
	c.Cli = cli
	return err
}

// setup ssh shell session
// set Session and StdinPipe here,
// and the Session.Stdout and Session.Sdterr are also set.
func (c *LinuxClient) NewSession(cols, rows int) error {
	sshSession, err := c.Cli.NewSession()
	if err != nil {
		return err
	}

	// we set stdin, then we can write data to ssh server via this stdin.
	// but, as for reading data from ssh server, we can set Session.Stdout and Session.Stderr
	// to receive data from ssh server, and write back to somewhere.
	stdinP, err := sshSession.StdinPipe()
	if err != nil {
		return err
	}
	stdoutP, err := sshSession.StdoutPipe()
	if err != nil {
		return err
	}

	comboWriter := new(wsBufferWriter)
	//ssh.stdout and stderr will write output into comboWriter
	sshSession.Stdout = comboWriter
	sshSession.Stderr = comboWriter

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,     // disable echo
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}
	// Request pseudo terminal
	if err := sshSession.RequestPty("xterm", rows, cols, modes); err != nil {
		return err
	}
	// Start remote shell
	if err := sshSession.Shell(); err != nil {
		return err
	}
	c.SshConn = &SshConn{StdinPipe: stdinP,StdoutPipe:stdoutP,Session: sshSession}
	return nil
}

func (c *LinuxClient) Close(){
	if c.IsClose != true{
		c.Record.File.Close()
		if err := c.SshConn.StdinPipe.Close();err != nil{
			log.Error("Close ssh connect stdin pipe error by",err)
		}else{
			log.Info("Close ssh connect stdin pipe ok")
		}
		if err := c.SshConn.Session.Close();err != nil{
			log.Error("Close ssh connect session error by",err)
		}else{
			log.Info("Close ssh connect session ok")
		}
		if err := c.Cli.Close();err != nil{
			log.Error("Close ssh client error by",err)
		}else{
			log.Info("Close ssh client ok")
		}
	}
	c.IsClose = true

}

