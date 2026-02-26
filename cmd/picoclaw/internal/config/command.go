package config

import (
	"github.com/spf13/cobra"
)

func NewConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration and test connections",
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(newTestCommand())
	cmd.AddCommand(newModelsCommand())
	cmd.AddCommand(newDiscoverCommand())

	return cmd
}
