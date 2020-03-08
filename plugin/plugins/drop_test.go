package plugins

import pg "github.com/bluemedora/bplogagent/plugin"

func newFakeNullOutput() *DropOutput {
	return &DropOutput{
		DefaultPlugin: pg.DefaultPlugin{
			PluginID:   "testnull",
			PluginType: "null",
		},
	}
}
