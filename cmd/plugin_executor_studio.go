//go:build plugin_studio
// +build plugin_studio

package cmd

import (
	"github.com/spf13/cobra"
)

func ExecuteStudioPlugin() error {
	studioRoot := &cobra.Command{
		Use:   "flash",
		Short: "FlashORM Studio Plugin",
	}

	studioRoot.AddCommand(studioCmd)

	return studioRoot.Execute()
}
