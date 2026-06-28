package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/google/uuid"
	"github.com/nossh/nossh/internal/agent"
	"github.com/nossh/nossh/internal/codes"
	"github.com/nossh/nossh/internal/config"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "status":
			runStatus()
			return
		case "init-config":
			runInitConfig()
			return
		}
	}

	configPath := flag.String("config", "/etc/nossh/agent.yaml", "agent config path")
	flag.Parse()

	cfg, err := config.LoadAgent(*configPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	if cfg.ServerURL == "" || cfg.InstallToken == "" {
		log.Fatal("server_url and install_token must be set in agent config")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Printf("nossh-agent starting code=%s server=%s", cfg.Code, cfg.ServerURL)
	if err := agent.New(cfg).Run(ctx); err != nil && err != context.Canceled {
		log.Fatalf("agent stopped: %v", err)
	}
}

func runStatus() {
	configPath := "/etc/nossh/agent.yaml"
	if len(os.Args) > 2 {
		configPath = os.Args[2]
	}
	cfg, err := config.LoadAgent(configPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	fmt.Printf("Agent code:  %s\n", cfg.Code)
	fmt.Printf("Agent UUID:  %s\n", cfg.UUID)
	fmt.Printf("Server:      %s\n", cfg.ServerURL)
}

func runInitConfig() {
	fs := flag.NewFlagSet("init-config", flag.ExitOnError)
	configPath := fs.String("config", "/etc/nossh/agent.yaml", "agent config path")
	server := fs.String("server", "", "bridge server host:port")
	token := fs.String("token", "", "install token")
	_ = fs.Parse(os.Args[2:])

	if *server == "" || *token == "" {
		fmt.Fprintln(os.Stderr, "usage: nossh-agent init-config --server HOST:4443 --token TOKEN [--config path]")
		os.Exit(1)
	}

	if _, err := os.Stat(*configPath); err == nil {
		fmt.Fprintf(os.Stderr, "config already exists: %s\n", *configPath)
		os.Exit(1)
	}

	code, err := codes.Generate()
	if err != nil {
		log.Fatalf("generate code: %v", err)
	}

	cfg := config.Agent{
		ServerURL:    *server,
		InstallToken: *token,
		Code:         code,
		UUID:         uuid.NewString(),
		SSHPort:      config.DefaultSSHPort,
	}
	if err := config.SaveAgent(*configPath, cfg); err != nil {
		log.Fatalf("save config: %v", err)
	}

	fmt.Println(code)
}
