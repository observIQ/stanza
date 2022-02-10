package main

import (
	// Load linux only packages when importing input operators
	_ "github.com/observiq/stanza/v2/operator/builtin/input/journald"
)
