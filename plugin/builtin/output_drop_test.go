package builtin

import (
	"github.com/bluemedora/bplogagent/plugin/base"
)

func newFakeNullOutput() *DropOutput {
	return &DropOutput{
		OutputPlugin: base.OutputPlugin{
			Plugin: base.Plugin{
				PluginID:   "testnull",
				PluginType: "null",
			},
		},
	}
}
