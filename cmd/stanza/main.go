package main

import (
	"os"
  _ "github.com/observiq/stanza/operator/builtin"
)

func main() {
	rootCmd := NewRootCmd()
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
