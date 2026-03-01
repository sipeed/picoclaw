package plugin

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/cmd/picoclaw/internal/pluginruntime"
)

const (
	formatText = "text"
	formatJSON = "json"
)

type pluginStatus struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

func newListCommand() *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List configured plugin status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if format != formatText && format != formatJSON {
				return fmt.Errorf("invalid value for --format: %q (allowed: %s, %s)", format, formatText, formatJSON)
			}

			cfg, err := internal.LoadConfig()
			if err != nil {
				return fmt.Errorf("error loading config: %w", err)
			}

			_, summary, err := pluginruntime.ResolveConfiguredPlugins(cfg)
			statuses := buildPluginStatuses(summary)

			if outputErr := renderPluginStatuses(cmd.OutOrStdout(), format, statuses); outputErr != nil {
				return outputErr
			}
			if err != nil {
				return fmt.Errorf("error resolving configured plugins: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&format, "format", formatText, "Output format (text|json)")

	return cmd
}

func buildPluginStatuses(summary pluginruntime.Summary) []pluginStatus {
	statuses := make([]pluginStatus, 0, len(summary.Enabled)+len(summary.Disabled)+len(summary.UnknownEnabled)+len(summary.UnknownDisabled))

	for _, name := range summary.Enabled {
		statuses = append(statuses, pluginStatus{Name: name, Status: "enabled"})
	}
	for _, name := range summary.Disabled {
		statuses = append(statuses, pluginStatus{Name: name, Status: "disabled"})
	}
	for _, name := range summary.UnknownEnabled {
		statuses = append(statuses, pluginStatus{Name: name, Status: "unknown-enabled"})
	}
	for _, name := range summary.UnknownDisabled {
		statuses = append(statuses, pluginStatus{Name: name, Status: "unknown-disabled"})
	}

	sort.Slice(statuses, func(i, j int) bool {
		if statuses[i].Name == statuses[j].Name {
			return statuses[i].Status < statuses[j].Status
		}
		return statuses[i].Name < statuses[j].Name
	})

	return statuses
}

func renderPluginStatuses(w io.Writer, format string, statuses []pluginStatus) error {
	switch format {
	case formatText:
		if _, err := fmt.Fprintln(w, "NAME\tSTATUS"); err != nil {
			return err
		}
		for _, status := range statuses {
			if _, err := fmt.Fprintf(w, "%s\t%s\n", status.Name, status.Status); err != nil {
				return err
			}
		}
		return nil
	case formatJSON:
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(statuses)
	default:
		return fmt.Errorf("invalid value for --format: %q (allowed: %s, %s)", format, formatText, formatJSON)
	}
}
