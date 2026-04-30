package freeride

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/pkg/tools"
)

// NewFreerideCommand returns a new cobra.Command for managing OpenRouter free models.
func NewFreerideCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "freeride",
		Short: "Manage OpenRouter free models and fallbacks",
		Long:  "FreeRide automatically discovers and configures OpenRouter's best free models as fallbacks for your PicoClaw agent.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(
		newListCommand(),
		newAutoCommand(),
		newStatusCommand(),
		newSetTimeoutCommand(),
	)

	return cmd
}

func newListCommand() *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available free models from OpenRouter",
		RunE: func(cmd *cobra.Command, args []string) error {
			cdPath := filepath.Join(internal.GetPicoclawHome(), "freeride_cooldown.json")
			t := tools.NewFreeRideTool(internal.GetConfigPath(), cdPath, nil)
			result := t.Execute(context.Background(), map[string]any{
				"command": "list",
				"limit":   float64(limit),
			})
			fmt.Println(result.ForLLM)
			return nil
		},
	}
	cmd.Flags().IntVarP(&limit, "limit", "l", 10, "Number of models to list")
	return cmd
}

func newAutoCommand() *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use:   "auto",
		Short: "Automatically configure best free models as fallbacks",
		RunE: func(cmd *cobra.Command, args []string) error {
			cdPath := filepath.Join(internal.GetPicoclawHome(), "freeride_cooldown.json")
			t := tools.NewFreeRideTool(internal.GetConfigPath(), cdPath, nil)
			result := t.Execute(context.Background(), map[string]any{
				"command": "auto",
				"limit":   float64(limit),
			})
			fmt.Println(result.ForLLM)
			return nil
		},
	}
	cmd.Flags().IntVarP(&limit, "limit", "l", 5, "Number of fallbacks to configure")
	return cmd
}

func newStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Check current FreeRide configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cdPath := filepath.Join(internal.GetPicoclawHome(), "freeride_cooldown.json")
			t := tools.NewFreeRideTool(internal.GetConfigPath(), cdPath, nil)
			result := t.Execute(context.Background(), map[string]any{
				"command": "status",
			})
			fmt.Println(result.ForLLM)
			return nil
		},
	}
}

func newSetTimeoutCommand() *cobra.Command {
	var timeout int
	cmd := &cobra.Command{
		Use:   "settimeout",
		Short: "Set request timeout for all OpenRouter models",
		RunE: func(cmd *cobra.Command, args []string) error {
			cdPath := filepath.Join(internal.GetPicoclawHome(), "freeride_cooldown.json")
			t := tools.NewFreeRideTool(internal.GetConfigPath(), cdPath, nil)
			result := t.Execute(context.Background(), map[string]any{
				"command": "settimeout",
				"timeout": float64(timeout),
			})
			fmt.Println(result.ForLLM)
			return nil
		},
	}
	cmd.Flags().IntVarP(&timeout, "timeout", "t", 300, "Request timeout in seconds (default 300)")
	return cmd
}
