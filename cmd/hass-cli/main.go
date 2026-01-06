// Package main is the entry point for hass-cli.
package main

import (
	"fmt"
	"os"

	"github.com/dorinclisu/hass-cli/internal/cli"
)

// Version is set at build time via ldflags
var Version = "dev"

func main() {
	cli.SetVersion(Version)
	if err := cli.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
