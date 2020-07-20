package input

import (
	// Load embedded packages when importing input plugins
	_ "github.com/observiq/carbon/plugin/builtin/input/eventlog"
	_ "github.com/observiq/carbon/plugin/builtin/input/file"
)
