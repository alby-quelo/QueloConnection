package configfile

import (
	"net"
	"os"
	"path/filepath"
	"strings"
)

// PortableRoot is the directory next to the portable executables (nossh / GUI).
func PortableRoot() (string, bool) {
	exe, err := os.Executable()
	if err != nil {
		return "", false
	}
	dir := filepath.Dir(exe)
	for _, name := range []string{"nossh.exe", "nossh", "quelo-connect.exe", "quelo-connect-gui-win.exe"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return dir, true
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "client.conf")); err == nil {
		return dir, true
	}
	return "", false
}

func configRoot() string {
	if dir, ok := PortableRoot(); ok {
		return filepath.Join(dir, ".nossh")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".nossh"
	}
	return filepath.Join(home, ".config", "nossh")
}

func SavedMachinesPath() string {
	return filepath.Join(configRoot(), "saved-machines")
}

// ClientConfigPath returns where client.conf should be read/written.
func ClientConfigPath() string {
	if dir, ok := PortableRoot(); ok {
		return filepath.Join(dir, "client.conf")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "client.conf"
	}
	return filepath.Join(home, ".config", "nossh", "client.conf")
}

// SaveClient writes host, port, token and machine to client.conf.
func SaveClient(cfg Client) error {
	path := ClientConfigPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	host := trim(cfg.Host)
	if host == "" && trim(cfg.Server) != "" {
		if h, p, err := net.SplitHostPort(trim(cfg.Server)); err == nil {
			host = h
			if trim(cfg.Port) == "" {
				cfg.Port = p
			}
		} else {
			host = trim(cfg.Server)
		}
	}
	port := trim(cfg.Port)
	if port == "" {
		port = DefaultClientPort
	}

	var b strings.Builder
	b.WriteString("# nossh client configuration\n")
	b.WriteString("host=")
	b.WriteString(host)
	b.WriteString("\nport=")
	b.WriteString(port)
	b.WriteString("\ntoken=")
	b.WriteString(trim(cfg.Token))
	b.WriteString("\nmachine=")
	b.WriteString(trim(cfg.Machine))
	b.WriteString("\n")
	return os.WriteFile(path, []byte(b.String()), 0o600)
}

// ResetClient removes client.conf, saved machines and known_hosts.
func ResetClient() error {
	_ = os.Remove(ClientConfigPath())
	_ = os.Remove(SavedMachinesPath())
	_ = os.Remove(filepath.Join(configRoot(), "known_hosts"))
	return nil
}
