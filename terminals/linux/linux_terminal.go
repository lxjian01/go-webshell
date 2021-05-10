package linux

import (
	"bytes"
	"encoding/json"
	"github.com/gorilla/websocket"
	globalConf "go-webshell/global/config"
	"go-webshell/global/log"
	"go-webshell/terminals"
	"golang.org/x/crypto/ssh"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
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

type LinuxTerminal struct {
	terminals.BaseTerminal
	Host string
	Cli *ssh.Client
	SshConn *SshConn
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

func publicKeyAuthFunc(singer ssh.Signer) ssh.AuthMethod{
	return ssh.PublicKeys(singer)
}

func NewLinuxTerminal(w http.ResponseWriter, r *http.Request, responseHeader http.Header, host string) (*LinuxTerminal, error) {
	// 初始化websocket
	wsConn, err := terminals.NewWebsocket(w, r, responseHeader)
	if err != nil {
		log.Error("Init websocket error by",err)
		return nil, err
	}
	log.Info("Websocket connect ok")
	var c LinuxTerminal
	c.Host = host
	c.WsConn = wsConn
	LinuxUser := globalConf.GetAppConfig().LinuxUser
	config := &ssh.ClientConfig{
		Timeout:         time.Second * 5,
		User:            LinuxUser,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //这个可以， 但是不够安全
		//HostKeyCallback: hostKeyCallBackFunc(h.Host),
	}
	singer, err := GetSshSigner()
	if err != nil{
		return nil, err
	}
	config.Auth = []ssh.AuthMethod{publicKeyAuthFunc(singer)}
	addr := net.JoinHostPort(c.Host, strconv.Itoa(22))
	cli, err1 := ssh.Dial("tcp", addr, config)
	c.Cli = cli
	return &c, err1
}

// setup ssh shell session
// set Session and StdinPipe here,
// and the Session.Stdout and Session.Sdterr are also set.
func (c *LinuxTerminal) NewSession(cols, rows int) error {
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

func (c *LinuxTerminal) LinuxReadWebsocketWrite(){
	for {
		// linux reader and websocket writer
		buf := make([]byte, 1024)
		n, err := c.SshConn.StdoutPipe.Read(buf)
		if err != nil {
			log.Error("Read docker message error by ",err)
			c.Close()
			return
		}
		cmd := string(buf[:n])
		c.WriteRecord(cmd)
		err1 := c.WsConn.WriteMessage(websocket.BinaryMessage, buf)
		if err1 != nil {
			log.Error("Docker message write to websocket error by ",err1)
			return
		}
	}
}

func (c *LinuxTerminal) LinuxWriteWebsocketRead(userCode string){
	var build strings.Builder
	for {
		// linux writer and websocket reader
		_, p, err := c.WsConn.ReadMessage()
		if err != nil {
			log.Error("Read websocket message error by ",err)
			c.Close()
			return
		}
		cmd := string(p)
		if strings.HasPrefix(cmd, "{\"type\":\"resize\",\"rows\":"){
			var resizeParams terminals.ResizeParams
			if err := json.Unmarshal([]byte(cmd),&resizeParams);err != nil{
				log.Error("Unmarshal resize params error by ",err)
			}
			if err := c.SshConn.Session.WindowChange(resizeParams.Rows,resizeParams.Cols);err != nil{
				log.Error("Change ssh windows size error by ",err)
			}
		}else{
			terminals.WriteCmdLog(&build, cmd, userCode, c.Host,1)
			_,err1  := c.SshConn.StdinPipe.Write(p)
			if err1 != nil {
				log.Error("Websocket message copy to docker error by ",err)
				return
			}
		}
	}
}

func (c *LinuxTerminal) Close(){
	if c.Cli != nil {
		// close linux client
		if err := c.Cli.Close();err != nil{
			log.Error("Close ssh client error by",err)
		}else{
			log.Info("Close ssh client ok")
		}

		// close ssh connection
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
	}
	c.CloseWs()
	c.CloseRecordFile()
}