package kubernetes

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"go-webshell/global/log"
	"go-webshell/terminals"
	"io"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	"strings"
)

// PtyHandler is what remotecommand expects from a pty
type PtyHandler interface {
	remotecommand.TerminalSizeQueue
	Done()
	Tty() bool
	Stdin() io.Reader
	Stdout() io.Writer
	Stderr() io.Writer
}

// TerminalSession implements PtyHandler
type TerminalSession struct {
	wsConn   *websocket.Conn
	sizeChan chan remotecommand.TerminalSize
	doneChan chan struct{}
	tty      bool
}

// NewTerminalSession create TerminalSession
func NewTerminalSession(ws *websocket.Conn) (*TerminalSession, error) {
	session := &TerminalSession{
		wsConn:   ws,
		tty:      true,
		sizeChan: make(chan remotecommand.TerminalSize),
		doneChan: make(chan struct{}),
	}
	return session, nil
}

// Exec exec into a pod
func Exec(ptyHandler PtyHandler, namespace, podName string) error {
	req := GetClientset().CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec")
	cmd := []string{
		"/bin/sh",
		"-c",
		"TERM=xterm-256color; export TERM; /bin/bash"}
	req.VersionedParams(&v1.PodExecOptions{
		Stdin:     true,
		Stdout:    true,
		Stderr:    true,
		TTY:       true,
		// Container: containerName,
		Command:   cmd,
	}, scheme.ParameterCodec)

	executor, err := remotecommand.NewSPDYExecutor(GetConfig(), "POST", req.URL())
	if err != nil {
		return err
	}
	err = executor.Stream(remotecommand.StreamOptions{
		Stdin:             ptyHandler.Stdin(),
		Stdout:            ptyHandler.Stdout(),
		Stderr:            ptyHandler.Stderr(),
		TerminalSizeQueue: ptyHandler,
		Tty:               ptyHandler.Tty(),
	})
	return err
}

// Next called in a loop from remotecommand as long as the process is running
func (t *TerminalSession) Next() *remotecommand.TerminalSize {
	select {
	case size := <-t.sizeChan:
		return &size
	case <-t.doneChan:
		return nil
	}
}

// Done done, must call Done() before connection close, or Next() would not exits.
func (t *TerminalSession) Done() {
	close(t.doneChan)
}

// Tty ...
func (t *TerminalSession) Tty() bool {
	return t.tty
}

// Stdin ...
func (t *TerminalSession) Stdin() io.Reader {
	return t
}

// Stdout ...
func (t *TerminalSession) Stdout() io.Writer {
	return t
}

// Stderr ...
func (t *TerminalSession) Stderr() io.Writer {
	return t
}

// Close close session
func (t *TerminalSession) Close() error {
	return t.wsConn.Close()
}

// Read called in a loop from remotecommand as long as the process is running
func (t *TerminalSession) Read(p []byte) (int, error) {
	_, message, err := t.wsConn.ReadMessage()
	if err != nil {
		log.Error("Read websocket message error by",err)
		t.Close()
		return 0, nil
	}
	cmd := string(message)
	if strings.HasPrefix(cmd, "{\"type\":\"resize\",\"rows\":"){
		var resizeParams terminals.ResizeParams
		if err := json.Unmarshal(message,&resizeParams);err != nil{
			log.Error("Unmarshal resize params error by",err)
		}
		height := uint16(resizeParams.Rows)
		width := uint16(resizeParams.Cols)
		t.sizeChan <- remotecommand.TerminalSize{Width: width, Height: height}
		return 0, nil

	}else {
		return copy(p, message), nil
	}
}

// Write called from remotecommand whenever there is any output
func (t *TerminalSession) Write(p []byte) (int, error) {
	if err := t.wsConn.WriteMessage(websocket.TextMessage, p); err != nil {
		log.Warnf("write message err: %v \n", err)
		t.Close()
		return 0, err
	}
	return len(p), nil
}

