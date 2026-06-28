package webterm

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
)

type resizeMsg struct {
	Type string `json:"type"`
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
}

// RunPTY bridges a WebSocket to cmd's PTY (raw terminal I/O).
func RunPTY(conn *websocket.Conn, cmd *exec.Cmd) error {
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return err
	}
	defer func() { _ = ptmx.Close() }()

	var wg sync.WaitGroup
	done := make(chan struct{})

	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 32*1024)
		for {
			n, err := ptmx.Read(buf)
			if n > 0 {
				if werr := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); werr != nil {
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(done)
		for {
			mt, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			switch mt {
			case websocket.BinaryMessage, websocket.TextMessage:
				if len(data) > 0 && data[0] == '{' {
					var msg resizeMsg
					if json.Unmarshal(data, &msg) == nil && msg.Type == "resize" && msg.Cols > 0 && msg.Rows > 0 {
						_ = pty.Setsize(ptmx, &pty.Winsize{Cols: msg.Cols, Rows: msg.Rows})
						continue
					}
				}
				if _, werr := ptmx.Write(data); werr != nil {
					return
				}
			}
		}
	}()

	wg.Wait()
	if err := cmd.Wait(); err != nil {
		log.Printf("webterm: session ended: %v", err)
	}
	return nil
}

// DiscardReader drains and closes r (used when cmd fails to start).
func DiscardReader(r io.ReadCloser) {
	if r == nil {
		return
	}
	_, _ = io.Copy(io.Discard, r)
	_ = r.Close()
}
