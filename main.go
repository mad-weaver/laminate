package main

import (
	"log/slog"
	"os"

	cmd "github.com/mad-weaver/laminate/cmd/laminate"
)

// Main function.
func main() {
	app := cmd.NewApp()
	if app == nil {
		slog.Error("Failed to initialize app")
		os.Exit(1)
	}

	if err := app.Run(os.Args); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}
