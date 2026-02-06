//go:build plugin_studio
// +build plugin_studio

package main

import (
	"fmt"
	"os"

	"github.com/Lumos-Labs-HQ/flash/cmd"
)

func main() {
	// This is the studio plugin binary
	if err := cmd.ExecuteStudioPlugin(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
