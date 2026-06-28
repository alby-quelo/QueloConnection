package client

import (
	"os"
	"path/filepath"
)

// PortableDir returns the directory next to nossh.exe when client.conf is present there.
func PortableDir() (string, bool) {
	exe, err := os.Executable()
	if err != nil {
		return "", false
	}
	dir := filepath.Dir(exe)
	if _, err := os.Stat(filepath.Join(dir, "client.conf")); err == nil {
		return dir, true
	}
	return "", false
}

func stateDir() string {
	if dir, ok := PortableDir(); ok {
		d := filepath.Join(dir, ".nossh")
		_ = os.MkdirAll(d, 0o700)
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	d := filepath.Join(home, ".nossh")
	_ = os.MkdirAll(d, 0o700)
	return d
}
