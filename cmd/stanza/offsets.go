package main

import (
	"fmt"
	"io"
	"os"

	"github.com/observiq/stanza/v2/database"
	"github.com/observiq/stanza/v2/operator/helper"
	"github.com/spf13/cobra"
	"go.etcd.io/bbolt"
	"go.uber.org/zap"
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
	var all bool

	offsetsClear := &cobra.Command{
		Use:   "clear [flags] [operator_ids]",
		Short: "Clear persisted offsets from the database",
		Args:  cobra.ArbitraryArgs,
		Run: func(command *cobra.Command, args []string) {
			db, err := database.OpenDatabase(rootFlags.DatabaseFile)
			exitOnErr("Failed to open database", err)
			defer db.Close()
			defer func() { _ = db.Sync() }()

			if all {
				if len(args) != 0 {
					_, err := stdout.Write([]byte("Providing a list of operator IDs does nothing with the --all flag\n"))
					if err != nil {
						exitOnErr("", err)
					}
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
					_, err := stdout.Write([]byte("Must either specify a list of operators or the --all flag\n"))
					if err != nil {
						exitOnErr("", err)
					}
					os.Exit(1)
				}

				for _, operatorID := range args {
					err = db.Update(func(tx *bbolt.Tx) error {
						offsetBucket := tx.Bucket(helper.OffsetsBucket)
						if offsetBucket == nil {
							return nil
						}

						return offsetBucket.DeleteBucket([]byte(operatorID))
					})
					exitOnErr("Failed to delete offsets", err)
				}
			}
		},
	}

	offsetsClear.Flags().BoolVar(&all, "all", false, "clear offsets for all inputs")

	return offsetsClear
}

// NewOffsetsListCmd returns the command for listing offsets
func NewOffsetsListCmd(rootFlags *RootFlags) *cobra.Command {
	offsetsList := &cobra.Command{
		Use:   "list",
		Short: "List operators with persisted offsets",
		Args:  cobra.NoArgs,
		Run: func(command *cobra.Command, args []string) {
			db, err := database.OpenDatabase(rootFlags.DatabaseFile)
			exitOnErr("Failed to open database", err)
			defer db.Close()

			err = db.View(func(tx *bbolt.Tx) error {
				offsetBucket := tx.Bucket(helper.OffsetsBucket)
				if offsetBucket == nil {
					return nil
				}

				return offsetBucket.ForEach(func(key, value []byte) error {
					_, err := stdout.Write(append(key, '\n'))
					return err
				})
			})
			if err != nil {
				exitOnErr("Failed to read database", err)
			}
		},
	}

	return offsetsList
}

func exitOnErr(msg string, err error) {
	var sugaredLogger *zap.SugaredLogger
	if err != nil {
		_, err := os.Stderr.WriteString(fmt.Sprintf("%s: %s\n", msg, err))
		if err != nil {
			sugaredLogger.Errorw("Failed to write to stdout", zap.Any("error", err))
		}
		os.Exit(1)
	}
}
