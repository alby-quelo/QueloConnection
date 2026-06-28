//go:build windows

package launcher

import (
	"fmt"
	"os/exec"
	"syscall"
)

const createNoWindow = 0x08000000

// StartNosshConnect opens one console for nossh connect.
// nossh must be started via "start /wait" so stdin/stdout attach to the new
// console; launching it directly from a GUI process leaves SSH without a TTY.
func StartNosshConnect(nossh, server, machine, user string) (*exec.Cmd, error) {
	proc := exec.Command("cmd.exe", "/c", "start", "/wait", "Quelo Connect SSH", nossh, "-server", server, "connect", machine, user)
	proc.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: createNoWindow,
	}
	if err := proc.Start(); err != nil {
		return nil, fmt.Errorf("impossibile aprire il terminale: %w", err)
	}
	return proc, nil
}
