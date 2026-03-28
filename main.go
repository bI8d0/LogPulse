package main

import (
	"fmt"
	"os"

	"LogPulse/internal/config"
	"LogPulse/internal/ui"
)

func main() {
	cfg := config.Load()

	app := ui.NewApp(cfg)
	if err := app.Run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to start LogPulse: %v\n", err)
		os.Exit(1)
	}
}
