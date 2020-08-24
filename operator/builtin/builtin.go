package builtin

import (
	// Load embedded packages when importing builtin operators
	_ "github.com/observiq/stanza/operator/builtin/input"
	_ "github.com/observiq/stanza/operator/builtin/output"
	_ "github.com/observiq/stanza/operator/builtin/parser"
	_ "github.com/observiq/stanza/operator/builtin/transformer"
)
