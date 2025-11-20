//go:build plugins
// +build plugins

package main

import (
	"fmt"
	"os"

	"github.com/Lumos-Labs-HQ/flash/cmd"
)

func main() {
	// This is the 'all' plugin binary
	// It includes all features: core ORM + studio
	if err := cmd.ExecuteAllPlugin(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
