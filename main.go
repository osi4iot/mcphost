package main

import "github.com/osi4iot/mcphost/cmd"

var version = "dev"

func main() {
	cmd.Execute(version)
}
