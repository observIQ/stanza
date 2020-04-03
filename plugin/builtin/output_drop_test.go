package builtin

import (
	"github.com/bluemedora/bplogagent/plugin/helper"
)

func newFakeNullOutput() *DropOutput {
	return &DropOutput{
		BasicIdentity: helper.BasicIdentity{
			PluginID:   "testnull",
			PluginType: "null",
		},
	}
}
