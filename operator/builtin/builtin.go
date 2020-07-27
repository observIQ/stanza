package builtin

import (
	// Load embedded packages when importing builtin operators
	_ "github.com/observiq/carbon/operator/builtin/input"
	_ "github.com/observiq/carbon/operator/builtin/output"
	_ "github.com/observiq/carbon/operator/builtin/parser"
	_ "github.com/observiq/carbon/operator/builtin/transformer"
)
