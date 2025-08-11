package main

import (
	"context"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/mark3labs/mcphost/cmd"
)

var version = "dev"

func main() {
	rootCmd := cmd.GetRootCommand(version)
	if err := fang.Execute(context.Background(), rootCmd); err != nil {
		os.Exit(1)
	}
}
