package main

import (
	"fmt"
	"os"

	"github.com/Lumos-Labs-HQ/flash/cmd"
)

func main() {
	cmd.RegisterBaseCommands()

	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
