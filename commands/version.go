package commands

import (
	"github.com/observiq/carbon/internal/version"
	"github.com/spf13/cobra"
)

func NewVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Args:  cobra.NoArgs,
		Short: "Print the carbon version",
		Run: func(_ *cobra.Command, _ []string) {
			if version.Version != "" {
				println(version.Version)
			} else {
				println(version.GitHash)
			}
		},
	}
}
