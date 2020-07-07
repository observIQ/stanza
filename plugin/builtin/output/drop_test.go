package output

import (
	"github.com/observiq/bplogagent/plugin/helper"
)

func newFakeNullOutput() *DropOutput {
	return &DropOutput{
		OutputPlugin: helper.OutputPlugin{
			BasicPlugin: helper.BasicPlugin{
				PluginID:   "testnull",
				PluginType: "drop_output",
			},
		},
	}
}
