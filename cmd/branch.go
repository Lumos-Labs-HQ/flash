package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/Lumos-Labs-HQ/flash/internal/branch"
	"github.com/Lumos-Labs-HQ/flash/internal/config"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var branchCmd = &cobra.Command{
	Use:   "branch",
	Short: "Manage database branches",
	Long:  `Create, switch, list, and compare database branches for isolated development.`,
}

var branchCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new branch from current branch",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		branchName := args[0]

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		manager, err := branch.NewManager(cfg)
		if err != nil {
			return err
		}
		defer manager.Close()

		currentBranch, err := manager.GetCurrentBranch()
		if err != nil {
			return err
		}

		force, _ := cmd.Flags().GetBool("force")
		if !force {
			color.Yellow("⚠️  This will copy all schema and data from '%s' to '%s'.", currentBranch, branchName)
			fmt.Print("Continue? (y/N): ")
			var response string
			fmt.Scanln(&response)
			if response != "y" && response != "Y" {
				color.Red("✗ Cancelled")
				return nil
			}
		}

		color.Cyan("Creating branch '%s'...", branchName)
		
		ctx := context.Background()
		if err := manager.CreateBranch(ctx, branchName); err != nil {
			return fmt.Errorf("failed to create branch: %w", err)
		}

		color.Green("✓ Branch '%s' created successfully", branchName)
		return nil
	},
}

var branchSwitchCmd = &cobra.Command{
	Use:   "switch <name>",
	Short: "Switch to a different branch",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		branchName := args[0]

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		manager, err := branch.NewManager(cfg)
		if err != nil {
			return err
		}
		defer manager.Close()

		ctx := context.Background()
		if err := manager.SwitchBranch(ctx, branchName); err != nil {
			return fmt.Errorf("failed to switch branch: %w", err)
		}

		color.Green("✓ Switched to branch '%s'", branchName)
		return nil
	},
}

var branchListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all branches",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		manager, err := branch.NewManager(cfg)
		if err != nil {
			return err
		}
		defer manager.Close()

		branches, current, err := manager.ListBranches()
		if err != nil {
			return err
		}

		if len(branches) == 0 {
			color.Yellow("No branches found")
			return nil
		}

		fmt.Println()
		for _, b := range branches {
			prefix := "  "
			if b.Name == current {
				prefix = color.GreenString("* ")
			}

			status := ""
			if b.IsDefault {
				status = color.CyanString(" (default)")
			} else if b.Name == current {
				status = color.GreenString(" (active)")
			}

			age := time.Since(b.CreatedAt)
			ageStr := formatDuration(age)

			fmt.Printf("%s%-15s %s - Created %s ago\n", prefix, b.Name, status, ageStr)
		}
		fmt.Println()

		return nil
	},
}

var branchStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current branch",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		manager, err := branch.NewManager(cfg)
		if err != nil {
			return err
		}
		defer manager.Close()

		current, err := manager.GetCurrentBranch()
		if err != nil {
			return err
		}

		color.Cyan("Current branch: %s", color.GreenString(current))
		return nil
	},
}

var branchDiffCmd = &cobra.Command{
	Use:   "diff <branch1> <branch2>",
	Short: "Show schema differences between two branches",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		branch1 := args[0]
		branch2 := args[1]

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		manager, err := branch.NewManager(cfg)
		if err != nil {
			return err
		}
		defer manager.Close()

		ctx := context.Background()
		diff, err := manager.GetSchemaDiff(ctx, branch1, branch2)
		if err != nil {
			return fmt.Errorf("failed to get schema diff: %w", err)
		}

		if diff.IsEmpty() {
			color.Green("✓ No differences found between '%s' and '%s'", branch1, branch2)
			return nil
		}

		color.Cyan("\nSchema differences between '%s' and '%s':\n", branch1, branch2)
		fmt.Println(diff.String())

		return nil
	},
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		mins := int(d.Minutes())
		if mins == 1 {
			return "1 minute"
		}
		return fmt.Sprintf("%d minutes", mins)
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		if hours == 1 {
			return "1 hour"
		}
		return fmt.Sprintf("%d hours", hours)
	}
	days := int(d.Hours() / 24)
	if days == 1 {
		return "1 day"
	}
	return fmt.Sprintf("%d days", days)
}

func init() {
	rootCmd.AddCommand(branchCmd)
	
	branchCmd.AddCommand(branchCreateCmd)
	branchCmd.AddCommand(branchSwitchCmd)
	branchCmd.AddCommand(branchListCmd)
	branchCmd.AddCommand(branchStatusCmd)
	branchCmd.AddCommand(branchDiffCmd)

	branchCreateCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
}
