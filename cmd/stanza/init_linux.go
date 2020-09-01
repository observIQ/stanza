package main

import (
	// Load linux only packages when importing input operators
	_ "github.com/observiq/stanza/operator/builtin/input/journald"
)
