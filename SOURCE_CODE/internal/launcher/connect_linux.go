//go:build linux

package launcher

import (
	"fmt"
	"os/exec"
)

// StartNosshConnect runs nossh connect in the user's terminal emulator.
func StartNosshConnect(nossh, server, machine, user string) (*exec.Cmd, error) {
	cmd := fmt.Sprintf("%s -server %s connect %s %s",
		shellQuote(nossh), shellQuote(server), shellQuote(machine), shellQuote(user))
	return StartInTerminal(cmd)
}
