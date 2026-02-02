package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/johnrirwin/flyingforge/internal/app"
	"github.com/johnrirwin/flyingforge/internal/config"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Create application
	application, err := app.New(cfg)
	if err != nil {
		os.Exit(1)
	}

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		application.Logger.Info("Shutting down...")
		cancel()
		application.Shutdown(context.Background())
	}()

	// Run application
	if err := application.Run(ctx); err != nil && err != context.Canceled {
		application.Logger.Error("Application error", nil)
		os.Exit(1)
	}
}
