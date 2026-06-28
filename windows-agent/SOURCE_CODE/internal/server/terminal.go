package server

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"

	"github.com/gorilla/websocket"
	"github.com/nossh/nossh/internal/webterm"
)

var machineNameRe = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

var terminalUpgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (s *Server) handleTerminalWS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	machine := r.URL.Query().Get("machine")
	token := r.URL.Query().Get("token")
	if machine == "" || !machineNameRe.MatchString(machine) {
		http.Error(w, "invalid machine", http.StatusBadRequest)
		return
	}
	if err := webterm.ValidateToken(token, machine, s.cfg.AdminToken); err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if _, err := s.reg.GetActiveByNameOrCode(machine); err != nil {
		http.Error(w, "machine not available", http.StatusBadRequest)
		return
	}

	nosshBin, err := exec.LookPath("nossh")
	if err != nil {
		http.Error(w, "nossh client not installed on bridge server", http.StatusServiceUnavailable)
		return
	}

	serverAddr := fmt.Sprintf("127.0.0.1:%d", s.cfg.ClientPort)
	conn, err := terminalUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("webterm upgrade: %v", err)
		return
	}
	defer conn.Close()

	cmd := exec.Command(nosshBin, "-server", serverAddr, "connect", machine)
	cmd.Dir = s.cfg.DataDir
	if home, err := os.UserHomeDir(); err == nil {
		cmd.Env = append(os.Environ(), "HOME="+home)
	}

	log.Printf("webterm: session start machine=%s", machine)
	if err := webterm.RunPTY(conn, cmd); err != nil {
		log.Printf("webterm: machine=%s err=%v", machine, err)
		_ = conn.WriteMessage(websocket.TextMessage, []byte("\r\n[sessione terminale terminata]\r\n"))
	}
}
