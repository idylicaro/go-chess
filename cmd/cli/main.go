package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/idylicaro/go-chess/internal/cli"
	"github.com/idylicaro/go-chess/internal/debug"
)

func main() {
	// Initialize debug mode from environment variables
	_ = debug.EnableFromEnvironment()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	code := cli.Run(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
}
