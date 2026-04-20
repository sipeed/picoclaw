package freeride

import (
	"context"
	"fmt"

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
	)

	return cmd
}

func newListCommand() *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available free models from OpenRouter",
		RunE: func(cmd *cobra.Command, args []string) error {
			t := tools.NewFreeRideTool(internal.GetConfigPath(), nil)
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
			t := tools.NewFreeRideTool(internal.GetConfigPath(), nil)
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
			t := tools.NewFreeRideTool(internal.GetConfigPath(), nil)
			result := t.Execute(context.Background(), map[string]any{
				"command": "status",
			})
			fmt.Println(result.ForLLM)
			return nil
		},
	}
}
