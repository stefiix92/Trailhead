package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/michalstefanec/trailhead/internal/config"
	"github.com/michalstefanec/trailhead/internal/mcp"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	cfg, err := config.FromEnv()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "trailhead-mcp: config error: %v\n", err)
		os.Exit(2)
	}
	if err := mcp.Run(ctx, &cfg, os.Stdin, os.Stdout); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "trailhead-mcp: %v\n", err)
		os.Exit(1)
	}
}
