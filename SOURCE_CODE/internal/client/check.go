package client

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"
)

// CheckResult describes the outcome of a bridge lookup.
type CheckResult struct {
	Verified      bool
	SkipReason    string
	ServerMessage string
}

// CheckMachine verifies the name exists on the bridge server (active registration).
func CheckMachine(serverAddr, machineName string) error {
	res := ProbeMachine(serverAddr, machineName)
	if res.Verified {
		return nil
	}
	if res.SkipReason != "" {
		return fmt.Errorf("%s", res.SkipReason)
	}
	return translateCheckError(res.ServerMessage)
}

// ProbeMachine checks the machine on the bridge and reports whether verification succeeded.
func ProbeMachine(serverAddr, machineName string) CheckResult {
	machineName = strings.TrimSpace(machineName)
	if machineName == "" {
		return CheckResult{SkipReason: "nome macchina vuoto"}
	}

	res := probeCheck(serverAddr, machineName)
	if res.Verified {
		return res
	}
	if res.ServerMessage != "" && !isUnsupportedCheck(res.ServerMessage) {
		return res
	}

	// Older bridge servers may not implement CHECK; CONNECT still distinguishes
	// unknown machines from registered ones (including offline agents).
	return probeConnect(serverAddr, machineName)
}

func probeCheck(serverAddr, machineName string) CheckResult {
	conn, err := net.DialTimeout("tcp", serverAddr, 8*time.Second)
	if err != nil {
		return CheckResult{SkipReason: "server ponte non raggiungibile"}
	}
	defer conn.Close()

	if _, err := fmt.Fprintf(conn, "CHECK %s\n", machineName); err != nil {
		return CheckResult{SkipReason: err.Error()}
	}

	line, err := readResponseLine(conn)
	if err != nil {
		return CheckResult{SkipReason: "risposta del server ponte non valida"}
	}
	if strings.HasPrefix(line, "OK") {
		return CheckResult{Verified: true}
	}
	return CheckResult{ServerMessage: line}
}

func probeConnect(serverAddr, machineName string) CheckResult {
	conn, err := net.DialTimeout("tcp", serverAddr, 8*time.Second)
	if err != nil {
		return CheckResult{SkipReason: "server ponte non raggiungibile"}
	}
	defer conn.Close()

	if _, err := fmt.Fprintf(conn, "CONNECT %s\n", machineName); err != nil {
		return CheckResult{SkipReason: err.Error()}
	}

	line, err := readResponseLine(conn)
	if err != nil {
		return CheckResult{SkipReason: "risposta del server ponte non valida"}
	}
	if strings.HasPrefix(line, "OK") {
		return CheckResult{Verified: true}
	}

	raw := strings.TrimSpace(strings.TrimPrefix(line, "ERR "))
	lower := strings.ToLower(raw)
	if strings.Contains(lower, "offline") {
		// Registered on the bridge but agent not connected right now.
		return CheckResult{Verified: true}
	}
	return CheckResult{ServerMessage: line}
}

func readResponseLine(conn net.Conn) (string, error) {
	conn.SetReadDeadline(time.Now().Add(8 * time.Second))
	defer conn.SetReadDeadline(time.Time{})

	line, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func isUnsupportedCheck(line string) bool {
	raw := strings.TrimSpace(strings.TrimPrefix(line, "ERR "))
	return strings.EqualFold(raw, "invalid handshake")
}

// CheckErrorFromResult turns a failed probe into a user-facing error.
func CheckErrorFromResult(res CheckResult) error {
	if res.Verified {
		return nil
	}
	if res.SkipReason != "" {
		return fmt.Errorf("%s", res.SkipReason)
	}
	return translateCheckError(res.ServerMessage)
}

func translateCheckError(line string) error {
	raw := strings.TrimSpace(strings.TrimPrefix(line, "ERR "))
	lower := strings.ToLower(raw)

	switch {
	case raw == "":
		return fmt.Errorf("macchina non presente sul server ponte")
	case lower == "invalid handshake":
		return fmt.Errorf("server ponte troppo vecchio: aggiorna nossh-server per verificare le macchine")
	case strings.Contains(lower, "not active"):
		return fmt.Errorf("macchina registrata ma non ancora attivata sul ponte (assegna il nome lato admin)")
	case strings.Contains(lower, "not found"):
		return fmt.Errorf("macchina non trovata sul ponte: verifica il nome assegnato")
	case strings.Contains(lower, "not available"):
		return fmt.Errorf("macchina non presente sul server ponte")
	case strings.Contains(lower, "offline"):
		return fmt.Errorf("macchina registrata ma attualmente non raggiungibile")
	case strings.Contains(lower, "relay"):
		return fmt.Errorf("errore di collegamento al server ponte")
	default:
		return fmt.Errorf("verifica fallita sul ponte: %s", raw)
	}
}
