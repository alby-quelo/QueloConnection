package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	DefaultAgentPort  = 4443
	DefaultClientPort = 7000
	DefaultAdminPort  = 8081
	DefaultSSHPort    = 22
)

type Server struct {
	ListenAddr   string `yaml:"listen_addr"`
	AgentPort    int    `yaml:"agent_port"`
	ClientPort   int    `yaml:"client_port"`
	AdminPort    int    `yaml:"admin_port"`
	AdminToken   string `yaml:"admin_token"`
	InstallToken string `yaml:"install_token"`
	DataDir      string `yaml:"data_dir"`
}

type Agent struct {
	ServerURL    string `yaml:"server_url"`
	InstallToken string `yaml:"install_token"`
	Code         string `yaml:"code"`
	UUID         string `yaml:"uuid"`
	SSHPort      int    `yaml:"ssh_port"`
}

func DefaultServerConfig() Server {
	return Server{
		ListenAddr: "0.0.0.0",
		AgentPort:  DefaultAgentPort,
		ClientPort: DefaultClientPort,
		AdminPort:  DefaultAdminPort,
		DataDir:    "/var/lib/nossh",
	}
}

func LoadServer(path string) (Server, error) {
	cfg := DefaultServerConfig()
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("read server config: %w", err)
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse server config: %w", err)
	}
	if cfg.AgentPort == 0 {
		cfg.AgentPort = DefaultAgentPort
	}
	if cfg.ClientPort == 0 {
		cfg.ClientPort = DefaultClientPort
	}
	if cfg.AdminPort == 0 {
		cfg.AdminPort = DefaultAdminPort
	}
	if cfg.DataDir == "" {
		cfg.DataDir = "/var/lib/nossh"
	}
	return cfg, nil
}

func LoadAgent(path string) (Agent, error) {
	var cfg Agent
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("read agent config: %w", err)
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse agent config: %w", err)
	}
	if cfg.SSHPort == 0 {
		cfg.SSHPort = DefaultSSHPort
	}
	return cfg, nil
}

func SaveAgent(path string, cfg Agent) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal agent config: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write agent config: %w", err)
	}
	return nil
}
