package builtin

import (
	// Load embedded packages when importing builtin plugins
	_ "github.com/observiq/carbon/plugin/builtin/input"
	_ "github.com/observiq/carbon/plugin/builtin/output"
	_ "github.com/observiq/carbon/plugin/builtin/parser"
	_ "github.com/observiq/carbon/plugin/builtin/transformer"
)
