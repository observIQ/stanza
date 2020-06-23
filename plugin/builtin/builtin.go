package builtin

import (
	// Load embedded packages when importing builtin plugins
	_ "github.com/bluemedora/bplogagent/plugin/builtin/input"
	_ "github.com/bluemedora/bplogagent/plugin/builtin/output"
	_ "github.com/bluemedora/bplogagent/plugin/builtin/parser"
	_ "github.com/bluemedora/bplogagent/plugin/builtin/transformer"
)
