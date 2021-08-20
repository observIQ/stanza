package main

import (
	"fmt"
	"runtime"

	"github.com/observiq/stanza/version"
	"github.com/spf13/cobra"
)

// NewVersionCommand returns the cli command for version
func NewVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Args:  cobra.NoArgs,
		Short: "Print the stanza version",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Println(version.GetVersion(), runtime.GOOS, runtime.GOARCH)
		},
	}
}
