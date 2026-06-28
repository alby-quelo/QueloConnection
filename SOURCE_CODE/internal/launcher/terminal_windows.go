//go:build windows

package launcher

import (
	"fmt"
	"os/exec"
	"syscall"
)

// RunInTerminal starts cmd in a new Windows console window.
func RunInTerminal(cmd string) error {
	_, err := StartInTerminal(cmd)
	return err
}

// StartInTerminal opens cmd.exe /k with the given command and returns the starter process.
func StartInTerminal(cmd string) (*exec.Cmd, error) {
	inner := cmd + ` & echo. & echo Sessione terminata. Premi un tasto per chiudere... & pause >nul`
	proc := exec.Command("cmd.exe", "/c", "start", "/wait", "Quelo Connect SSH", "cmd.exe", "/k", inner)
	proc.Stdin = nil
	proc.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: createNoWindow,
	}
	if err := proc.Start(); err != nil {
		return nil, fmt.Errorf("impossibile aprire il terminale: %w", err)
	}
	return proc, nil
}
