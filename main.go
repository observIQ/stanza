package main

import (
	"os"

	"github.com/observiq/stanza/commands"
)

func main() {
	rootCmd := commands.NewRootCmd()
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
