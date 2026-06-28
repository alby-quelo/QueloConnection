//go:build linux

package launcher

import (
	"fmt"
	"os"
	"os/exec"
)

// NosshPath finds the nossh client binary.
func NosshPath() (string, error) {
	if p, err := exec.LookPath("nossh"); err == nil {
		return p, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("nossh non trovato nel PATH")
	}
	p := home + "/.local/bin/nossh"
	if _, err := os.Stat(p); err == nil {
		return p, nil
	}
	return "", fmt.Errorf("nossh non trovato: esegui prima install-client.sh")
}
