package commands

import (
	"fmt"
	"os"

	agent "github.com/bluemedora/bplogagent/agent"
	"github.com/bluemedora/bplogagent/plugin/helper"
	"github.com/spf13/cobra"
	"go.etcd.io/bbolt"
)

func NewOffsetsCmd(rootFlags *RootFlags) *cobra.Command {
	offsets := &cobra.Command{
		Use:   "offsets",
		Short: "Manage input plugin offsets",
		Args:  cobra.NoArgs,
		Run: func(command *cobra.Command, args []string) {
			println("No offsets subcommand specified. See `bplogagent offsets help` for details")
		},
	}

	offsets.AddCommand(NewOffsetsClearCmd(rootFlags))
	offsets.AddCommand(NewOffsetsListCmd(rootFlags))

	return offsets
}

func NewOffsetsClearCmd(rootFlags *RootFlags) *cobra.Command {
	var all bool

	offsetsClear := &cobra.Command{
		Use:   "clear [flags] [plugin_ids]",
		Short: "Clear persisted offsets from the database",
		Args:  cobra.ArbitraryArgs,
		Run: func(command *cobra.Command, args []string) {
			cfg, err := agent.NewConfigFromGlobs(rootFlags.ConfigFiles)
			exitOnErr("Failed to read configs from glob", err)
			cfg.SetDefaults(rootFlags.DatabaseFile, rootFlags.PluginDir)

			db, err := agent.OpenDatabase(cfg.DatabaseFile)
			exitOnErr("Failed to open database", err)
			defer db.Close()
			defer db.Sync()

			if all {
				if len(args) != 0 {
					println("Providing a list of plugin IDs does nothing with the --all flag")
				}

				err := db.Update(func(tx *bbolt.Tx) error {
					offsetsBucket := tx.Bucket(helper.OffsetsBucket)
					if offsetsBucket != nil {
						return tx.DeleteBucket(helper.OffsetsBucket)
					}
					return nil
				})
				exitOnErr("Failed to delete offsets", err)
			} else {
				if len(args) == 0 {
					println("Must either specify a list of plugins or the --all flag")
					os.Exit(1)
				}

				for _, pluginID := range args {
					err = db.Update(func(tx *bbolt.Tx) error {
						offsetBucket := tx.Bucket(helper.OffsetsBucket)
						if offsetBucket == nil {
							return nil
						}

						return offsetBucket.DeleteBucket([]byte(pluginID))
					})
					exitOnErr("Failed to delete offsets", err)
				}
			}
		},
	}

	offsetsClear.Flags().BoolVar(&all, "all", false, "clear offsets for all inputs")

	return offsetsClear
}

func NewOffsetsListCmd(rootFlags *RootFlags) *cobra.Command {
	offsetsList := &cobra.Command{
		Use:   "list",
		Short: "List plugins with persisted offsets",
		Args:  cobra.NoArgs,
		Run: func(command *cobra.Command, args []string) {
			cfg, err := agent.NewConfigFromGlobs(rootFlags.ConfigFiles)
			exitOnErr("Failed to read configs from glob", err)
			cfg.SetDefaults(rootFlags.DatabaseFile, rootFlags.PluginDir)

			db, err := agent.OpenDatabase(cfg.DatabaseFile)
			exitOnErr("Failed to open database", err)
			defer db.Close()

			db.View(func(tx *bbolt.Tx) error {
				offsetBucket := tx.Bucket(helper.OffsetsBucket)
				if offsetBucket == nil {
					return nil
				}

				return offsetBucket.ForEach(func(key, value []byte) error {
					println(string(key))
					return nil
				})
			})

		},
	}

	return offsetsList
}

func exitOnErr(msg string, err error) {
	if err != nil {
		os.Stderr.WriteString(fmt.Sprintf("%s: %s\n", msg, err))
		os.Exit(1)
	}
}
