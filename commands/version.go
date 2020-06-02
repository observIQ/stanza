package commands

import (
	"github.com/spf13/cobra"
)

var version string = "0.0.0"

func NewVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Args:  cobra.NoArgs,
		Short: "Print the bplogagent version",
		Run:   func(_ *cobra.Command, _ []string) { println(version) },
	}
}
