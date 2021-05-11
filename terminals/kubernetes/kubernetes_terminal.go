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
	"net/http"
	"strings"
)

var (
	// EndOfTransmission end
	EndOfTransmission = "\u0004"
	build strings.Builder
)

// PtyHandler is what remotecommand expects from a pty
type PtyHandler interface {
	remotecommand.TerminalSizeQueue
	Done()
	Stdin() io.Reader
	Stdout() io.Writer
	Stderr() io.Writer
}

// KubernetesTerminal implements PtyHandler
type KubernetesTerminal struct {
	terminals.BaseTerminal
	sizeChan chan remotecommand.TerminalSize
	doneChan chan struct{}
	tty      bool
	userCode string
	namespace string
	pod string
}

// create KubernetesTerminal
func NewKubernetesTerminal(w http.ResponseWriter, r *http.Request, responseHeader http.Header, userCode string, namespace string, pod string) (*KubernetesTerminal, error) {
	// 初始化websocket
	wsConn, err := terminals.NewWebsocket(w, r, responseHeader)
	if err != nil {
		log.Error("Init websocket error by",err)
		return nil, err
	}
	log.Info("Websocket connect ok")

	terminal := &KubernetesTerminal{
		tty:      true,
		sizeChan: make(chan remotecommand.TerminalSize),
		doneChan: make(chan struct{}),
		userCode: userCode,
		namespace: namespace,
		pod: pod,
	}
	terminal.WsConn = wsConn
	return terminal, nil
}

// Exec exec into a pod
func (t *KubernetesTerminal) CreateExec() error {
	req := GetClientset().CoreV1().RESTClient().Post().
		Resource("pods").
		Namespace(t.namespace).
		Name(t.pod).
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
		Stdin:             t.Stdin(),
		Stdout:            t.Stdout(),
		Stderr:            t.Stderr(),
		TerminalSizeQueue: t,
		Tty:               t.tty,
	})
	return err
}

// Next called in a loop from remotecommand as long as the process is running
func (t *KubernetesTerminal) Next() *remotecommand.TerminalSize {
	select {
	case size := <-t.sizeChan:
		return &size
	case <-t.doneChan:
		return nil
	}
}

// Done done, must call Done() before connection close, or Next() would not exits.
func (t *KubernetesTerminal) Done() {
	close(t.doneChan)
}

// Stdin ...
func (t *KubernetesTerminal) Stdin() io.Reader {
	return t
}

// Stdout ...
func (t *KubernetesTerminal) Stdout() io.Writer {
	return t
}

// Stderr ...
func (t *KubernetesTerminal) Stderr() io.Writer {
	return t
}

// Close close session
func (t *KubernetesTerminal) Close() {
	t.CloseWs()
	t.CloseRecordFile()
}

// Read called in a loop from remotecommand as long as the process is running
func (t *KubernetesTerminal) Read(p []byte) (int, error) {
	_, message, err := t.WsConn.ReadMessage()
	if err != nil {
		log.Error("Read websocket message error by",err)
		return copy(p, EndOfTransmission), err
	}
	cmd := string(message)
	if strings.HasPrefix(cmd, "{\"type\":\"resize\",\"rows\":"){
		var resizeParams terminals.ResizeParams
		if err := json.Unmarshal(message,&resizeParams);err != nil{
			log.Error("Unmarshal resize params error by",err)
			return copy(p, EndOfTransmission), err
		}
		height := uint16(resizeParams.Rows)
		width := uint16(resizeParams.Cols)
		t.sizeChan <- remotecommand.TerminalSize{Width: width, Height: height}
		return 0, nil

	}else {
		t.WriteCmdLog(&build, cmd, t.userCode, "c.Host", 0)
		return copy(p, message), nil
	}
}

// Write called from remotecommand whenever there is any output
func (t *KubernetesTerminal) Write(p []byte) (int, error) {
	//n := len(p)
	//cmd := string(p[:n])
	//t.WriteRecord(cmd)
	if err := t.WsConn.WriteMessage(websocket.TextMessage, p); err != nil {
		log.Warnf("write message err: %v \n", err)
		return copy(p, EndOfTransmission), err
	}
	return len(p), nil
}

