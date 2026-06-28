//go:build !windows

package main

import (
	"context"

	"github.com/nossh/nossh/internal/agent"
	"github.com/nossh/nossh/internal/config"
)

func runAgent(ctx context.Context, cfg config.Agent) error {
	return agent.New(cfg).Run(ctx)
}
