package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/observiq/stanza/v2/operator/helper/persist"
	"github.com/spf13/cobra"
)

var stdout io.Writer = os.Stdout

// NewOffsetsCmd returns the root command for managing offsets
func NewOffsetsCmd(rootFlags *RootFlags) *cobra.Command {
	offsets := &cobra.Command{
		Use:   "offsets",
		Short: "Manage input operator offsets",
		Args:  cobra.NoArgs,
	}

	offsets.AddCommand(NewOffsetsClearCmd(rootFlags))
	offsets.AddCommand(NewOffsetsListCmd(rootFlags))

	return offsets
}

// NewOffsetsClearCmd returns the command for clearing offsets
func NewOffsetsClearCmd(rootFlags *RootFlags) *cobra.Command {

	offsetsClear := &cobra.Command{
		Use:   "clear [flags] [operator_ids]",
		Short: "Clear persisted offsets from the database",
		Args:  cobra.ArbitraryArgs,
		Run: func(command *cobra.Command, args []string) {
			persister, err := persist.NewBBoltPersister(rootFlags.DatabaseFile)
			exitOnErr("Failed to open database", err)

			defer persister.Close()

			// Clear the database behind the bbolt persister
			if err := persister.Clear(); err != nil {
				exitOnErr("Failed to delete offsets", err)
			}
		},
	}

	return offsetsClear
}

// NewOffsetsListCmd returns the command for listing offsets
func NewOffsetsListCmd(rootFlags *RootFlags) *cobra.Command {
	offsetsList := &cobra.Command{
		Use:   "list",
		Short: "List operators with persisted offsets",
		Args:  cobra.NoArgs,
		Run: func(command *cobra.Command, args []string) {
			persister, err := persist.NewBBoltPersister(rootFlags.DatabaseFile)
			exitOnErr("Failed to open database", err)
			defer persister.Close()

			keys, err := persister.Keys()
			exitOnErr("Failed to read database", err)

			output := strings.Join(keys, "\n")
			stdout.Write([]byte(output))

			// Write out a final newline
			stdout.Write([]byte("\n"))
		},
	}

	return offsetsList
}

func exitOnErr(msg string, err error) {
	if err != nil {
		fmt.Printf("%s: %s\n", msg, err)
		os.Exit(1)
	}
}
