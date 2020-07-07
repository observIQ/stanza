package builtin

import (
	// Load embedded packages when importing builtin plugins
	_ "github.com/observiq/bplogagent/plugin/builtin/input"
	_ "github.com/observiq/bplogagent/plugin/builtin/output"
	_ "github.com/observiq/bplogagent/plugin/builtin/parser"
	_ "github.com/observiq/bplogagent/plugin/builtin/transformer"
)
