package commands

import (
	"github.com/observiq/carbon/internal/version"
	"github.com/spf13/cobra"
)

// NewVersionCommand returns the cli command for version
func NewVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Args:  cobra.NoArgs,
		Short: "Print the carbon version",
		Run: func(_ *cobra.Command, _ []string) {
			println(version.GetVersion())
		},
	}
}
