//go:build plugins
// +build plugins

package main

import (
	"fmt"
	"os"

	"github.com/Lumos-Labs-HQ/flash/cmd"
)

func main() {
	// This is the 'core' plugin binary
	// It includes all ORM features except studio
	if err := cmd.ExecuteCorePlugin(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
