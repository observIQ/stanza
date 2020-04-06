package builtin

import (
	"github.com/bluemedora/bplogagent/plugin/helper"
)

func newFakeNullOutput() *DropOutput {
	return &DropOutput{
		BasicPlugin: helper.BasicPlugin{
			PluginID:   "testnull",
			PluginType: "drop_output",
		},
	}
}
