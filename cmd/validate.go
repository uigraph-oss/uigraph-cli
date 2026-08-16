package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/uigraph-oss/uigraph-cli/pkg/config"
	"github.com/uigraph-oss/uigraph-cli/pkg/timeline"
)

var validateCmd = newValidateCmd()

func newValidateCmd() *cobra.Command {
	var path string
	cmd := &cobra.Command{
		Use:           "validate",
		Short:         "Validate local UiGraph artifacts",
		Long:          "Loads .uigraph.yaml and validates its schema, local references, and timeline sources without credentials or network access.",
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			if _, _, err := loadAndValidateArtifacts(path, false); err != nil {
				return fmt.Errorf("validation failed: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Valid UiGraph configuration: %s\n", path)
			return nil
		},
	}
	cmd.Flags().StringVar(&path, "config", ".uigraph.yaml", "Path to config file")
	return cmd
}

func loadAndValidateArtifacts(path string, requireMLEnvironment bool) (*config.Config, []timeline.Event, error) {
	cfg, err := config.Load(path)
	if err != nil {
		return nil, nil, fmt.Errorf("config: %w", err)
	}
	if requireMLEnvironment {
		if err := cfg.Validate(); err != nil {
			return nil, nil, fmt.Errorf("config: %w", err)
		}
	} else {
		if err := cfg.ValidateLocal(); err != nil {
			return nil, nil, fmt.Errorf("config: %w", err)
		}
	}
	events, err := timeline.Scan(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("timeline: %w", err)
	}
	return cfg, events, nil
}
