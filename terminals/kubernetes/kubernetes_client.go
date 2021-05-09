package kubernetes

import (
	"github.com/gorilla/websocket"
	"io"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
)

// PtyHandler
type PtyHandler interface {
	io.Reader
	io.Writer
	remotecommand.TerminalSizeQueue
}

func NewClient()  {
	// NewSPDYExecutor
	req := GetClientset().CoreV1().RESTClient().Post().
		Resource("pods").
		Name("podName").
		Namespace("namespace").
		SubResource("exec")
	req.VersionedParams(&v1.PodExecOptions{
		Container: "",
		Command: []string{"bash"},
		Stdin:     true,
		Stdout:    true,
		Stderr:    true,
		TTY:       true,
	}, scheme.ParameterCodec)
	//kubeconfig := GetConfig()
	//executor, err := remotecommand.NewSPDYExecutor(kubeconfig, "POST", req.URL())
	//if err != nil {
	//	log.Printf("NewSPDYExecutor err: %v", err)
	//	panic(err)
	//}
	//// 用IO读写替换 os stdout
	//ptyHandler := PtyHandler{
	//	os.Stdin,
	//	os.Stdout,
	//}
	//// Stream
	//err = executor.Stream(remotecommand.StreamOptions{
	//	Stdin:             ptyHandler,
	//	Stdout:            ptyHandler,
	//	Stderr:            ptyHandler,
	//	TerminalSizeQueue: ptyHandler,
	//	Tty:               true,
	//})
}

// TerminalSession
type TerminalSession struct {
	wsConn   *websocket.Conn
	sizeChan chan remotecommand.TerminalSize
	doneChan chan struct{}
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
//// Read called in a loop from remotecommand as long as the process is running
//func (t *TerminalSession) Read(p []byte) (int, error) {
//	_, message, err := t.wsConn.ReadMessage()
//	if err != nil {
//		log.Printf("read message err: %v", err)
//		return copy(p, webshell.EndOfTransmission), err
//	}
//	var msg webshell.TerminalMessage
//	if err := json.Unmarshal([]byte(message), &msg); err != nil {
//		log.Printf("read parse message err: %v", err)
//		// return 0, nil
//		return copy(p, webshell.EndOfTransmission), err
//	}
//	switch msg.Operation {
//	case "stdin":
//		return copy(p, msg.Data), nil
//	case "resize":
//		t.sizeChan <- remotecommand.TerminalSize{Width: msg.Cols, Height: msg.Rows}
//		return 0, nil
//	default:
//		log.Printf("unknown message type '%s'", msg.Operation)
//		// return 0, nil
//		return copy(p, webshell.EndOfTransmission), fmt.Errorf("unknown message type '%s'", msg.Operation)
//	}
//}
//
//// Write called from remotecommand whenever there is any output
//func (t *TerminalSession) Write(p []byte) (int, error) {
//	msg, err := json.Marshal(webshell.TerminalMessage{
//		Operation: "stdout",
//		Data:      string(p),
//	})
//	if err != nil {
//		log.Printf("write parse message err: %v", err)
//		return 0, err
//	}
//	if err := t.wsConn.WriteMessage(websocket.TextMessage, msg); err != nil {
//		log.Printf("write message err: %v", err)
//		return 0, err
//	}
//	return len(p), nil
//}

// Close close session
func (t *TerminalSession) Close() error {
	return t.wsConn.Close()
}