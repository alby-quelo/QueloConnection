//go:build linux

package launcher

import (
	"fmt"
	"os/exec"
	"strings"
	"syscall"
)

// RunInTerminal starts cmd in the user's default terminal emulator.
func RunInTerminal(cmd string) error {
	_, err := StartInTerminal(cmd)
	return err
}

// StartInTerminal starts the terminal and returns the process (Wait when it exits).
func StartInTerminal(cmd string) (*exec.Cmd, error) {
	inner := cmd + `; echo; echo 'Sessione terminata. Premi Invio per chiudere.'; read -r`

	candidates := [][]string{
		{"xdg-terminal-exec", "bash", "-c", inner},
		{"x-terminal-emulator", "-e", "bash", "-c", inner},
		{"gnome-terminal", "--", "bash", "-c", inner},
		{"konsole", "-e", "bash", "-c", inner},
		{"xfce4-terminal", "-e", "bash -c " + shellQuote(inner)},
		{"mate-terminal", "--", "bash", "-c", inner},
		{"xterm", "-e", "bash", "-c", inner},
	}

	for _, argv := range candidates {
		bin, err := exec.LookPath(argv[0])
		if err != nil {
			continue
		}
		argv[0] = bin
		proc := exec.Command(argv[0], argv[1:]...)
		proc.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
		proc.Stdin = nil
		if err := proc.Start(); err != nil {
			continue
		}
		return proc, nil
	}
	return nil, fmt.Errorf("nessun emulatore di terminale trovato")
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
