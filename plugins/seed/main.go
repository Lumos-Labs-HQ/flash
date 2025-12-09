//go:build plugins
// +build plugins

package main

import (
	"fmt"
	"os"

	"github.com/Lumos-Labs-HQ/flash/cmd"
)

func main() {
	if err := cmd.ExecuteSeedPlugin(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
