package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/nossh/nossh/internal/config"
	"github.com/nossh/nossh/internal/registry"
	"github.com/nossh/nossh/internal/server"
)

func main() {
	configPath := flag.String("config", "/etc/nossh/server.yaml", "server config path")
	flag.Parse()

	cfg, err := config.LoadServer(*configPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	if cfg.InstallToken == "" {
		log.Fatal("install_token must be set in server config")
	}

	reg, err := registry.Open(cfg.DataDir)
	if err != nil {
		log.Fatalf("registry: %v", err)
	}
	defer reg.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	srv := server.New(cfg, reg)
	log.Printf("nossh-server listening agent=:%d client=:%d admin=127.0.0.1:%d",
		cfg.AgentPort, cfg.ClientPort, cfg.AdminPort)

	if err := srv.Run(ctx); err != nil && err != context.Canceled {
		fmt.Fprintf(os.Stderr, "server stopped: %v\n", err)
		os.Exit(1)
	}
}
