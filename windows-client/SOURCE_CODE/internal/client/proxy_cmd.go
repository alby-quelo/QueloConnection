package client

import (
	"fmt"
	"os/exec"
	"runtime"
)

func proxyCommand(exe, serverAddr string) string {
	return fmt.Sprintf("%q -server %q proxy %%h", exe, serverAddr)
}

func sshExecutable() string {
	if runtime.GOOS != "windows" {
		return "ssh"
	}
	for _, name := range []string{"ssh.exe", "ssh"} {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	return "ssh"
}
