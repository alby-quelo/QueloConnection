package configfile

import (
	"net"
	"os"
	"path/filepath"
)

const DefaultClientPort = "7000"

type Client struct {
	Host    string
	Port    string
	Token   string
	Machine string
	// Server legacy: host:port in vecchi client.conf (server=...).
	Server string
}

func (c Client) ServerAddr() string {
	host := trim(c.Host)
	if host == "" && trim(c.Server) != "" {
		return trim(c.Server)
	}
	if host == "" {
		return ""
	}
	port := trim(c.Port)
	if port == "" {
		port = DefaultClientPort
	}
	if stringsContainsHostPort(host) {
		return host
	}
	return net.JoinHostPort(host, port)
}

func stringsContainsHostPort(host string) bool {
	if len(host) == 0 {
		return false
	}
	if host[0] == '[' {
		return true
	}
	for i := 0; i < len(host); i++ {
		if host[i] == ':' {
			return true
		}
	}
	return false
}

func LoadClient() Client {
	cfg := Client{
		Host:    os.Getenv("NOSSH_HOST"),
		Server:  os.Getenv("NOSSH_SERVER"),
		Machine: os.Getenv("NOSSH_MACHINE"),
		Token:   os.Getenv("NOSSH_TOKEN"),
		Port:    os.Getenv("NOSSH_PORT"),
	}

	for _, path := range clientConfigPaths() {
		if data, err := os.ReadFile(path); err == nil {
			cfg = mergeClientConf(cfg, string(data))
		}
	}
	return normalizeClient(cfg)
}

func normalizeClient(cfg Client) Client {
	if trim(cfg.Host) == "" && trim(cfg.Server) != "" {
		host, port, err := net.SplitHostPort(trim(cfg.Server))
		if err == nil {
			cfg.Host = host
			if trim(cfg.Port) == "" {
				cfg.Port = port
			}
		} else {
			cfg.Host = trim(cfg.Server)
		}
	}
	return cfg
}

func clientConfigPaths() []string {
	var paths []string
	paths = append(paths, ClientConfigPath())
	if home, err := os.UserHomeDir(); err == nil {
		p := filepath.Join(home, ".config", "nossh", "client.conf")
		if paths[0] != p {
			paths = append(paths, p)
		}
	}
	if home := os.Getenv("HOME"); home != "" {
		p := filepath.Join(home, ".config", "nossh", "client.conf")
		if paths[len(paths)-1] != p {
			paths = append(paths, p)
		}
	}
	return paths
}

func mergeClientConf(cfg Client, raw string) Client {
	for _, line := range splitLines(raw) {
		if k, v, ok := splitKV(line); ok {
			switch k {
			case "host":
				if trim(cfg.Host) == "" {
					cfg.Host = v
				}
			case "port":
				if trim(cfg.Port) == "" {
					cfg.Port = v
				}
			case "token":
				if trim(cfg.Token) == "" {
					cfg.Token = v
				}
			case "server":
				if trim(cfg.Server) == "" {
					cfg.Server = v
				}
			case "machine":
				if trim(cfg.Machine) == "" {
					cfg.Machine = v
				}
			}
		}
	}
	return cfg
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, trim(s[start:i]))
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, trim(s[start:]))
	}
	return out
}

func splitKV(line string) (string, string, bool) {
	line = trim(line)
	if line == "" || line[0] == '#' {
		return "", "", false
	}
	for i := 0; i < len(line); i++ {
		if line[i] == '=' {
			return trim(line[:i]), trim(line[i+1:]), true
		}
	}
	return "", "", false
}

func trim(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t' || s[0] == '\r') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}
