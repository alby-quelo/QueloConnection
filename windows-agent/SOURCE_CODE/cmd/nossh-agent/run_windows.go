//go:build windows

package main

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/nossh/nossh/internal/agent"
	"github.com/nossh/nossh/internal/config"
	"golang.org/x/sys/windows/svc"
)

type winAgentService struct {
	cfg config.Agent
}

func (m *winAgentService) Execute(_ []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (bool, uint32) {
	setupWindowsAgentLog()
	changes <- svc.Status{State: svc.StartPending}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		log.Printf("nossh-agent service starting code=%s server=%s", m.cfg.Code, m.cfg.ServerURL)
		errCh <- agent.New(m.cfg).Run(ctx)
	}()

	changes <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}

	for {
		select {
		case err := <-errCh:
			if err != nil && err != context.Canceled {
				log.Printf("agent stopped: %v", err)
				return false, 1
			}
			return false, 0
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				changes <- svc.Status{State: svc.StopPending}
				cancel()
				<-errCh
				return false, 0
			default:
				log.Printf("unexpected service control: %d", c.Cmd)
			}
		}
	}
}

func setupWindowsAgentLog() {
	dir := filepath.Join(os.Getenv("ProgramData"), "nossh")
	_ = os.MkdirAll(dir, 0o755)
	logPath := filepath.Join(dir, "agent.log")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	log.SetOutput(f)
	log.SetFlags(log.LstdFlags)
}

func runAgent(ctx context.Context, cfg config.Agent) error {
	isSvc, err := svc.IsWindowsService()
	if err != nil {
		return err
	}
	if !isSvc {
		return agent.New(cfg).Run(ctx)
	}

	// Registered Windows service (started by Service Control Manager).
	setupWindowsAgentLog()
	return svc.Run("nossh-agent", &winAgentService{cfg: cfg})
}
