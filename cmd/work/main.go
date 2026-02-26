package main

import (
	"os"

	"github.com/tSquaredd/work-cli/internal/commands"
)

// Set via ldflags: -ldflags "-X main.version=1.0.0"
var version = "dev"

func main() {
	commands.SetVersion(version)
	if err := commands.Execute(); err != nil {
		os.Exit(1)
	}
}
