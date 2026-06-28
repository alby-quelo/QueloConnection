//go:build windows

package launcher

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// NosshPath finds nossh.exe next to the GUI or on PATH.
func NosshPath() (string, error) {
	if exe, err := os.Executable(); err == nil {
		p := filepath.Join(filepath.Dir(exe), "nossh.exe")
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	if p, err := exec.LookPath("nossh.exe"); err == nil {
		return p, nil
	}
	if p, err := exec.LookPath("nossh"); err == nil {
		return p, nil
	}
	return "", fmt.Errorf("nossh.exe non trovato nella cartella portable")
}
