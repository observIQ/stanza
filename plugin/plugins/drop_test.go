package plugins

import pg "github.com/bluemedora/bplogagent/plugin"

func newFakeNullOutput() *DropOutput {
	return &DropOutput{
		DefaultPlugin: pg.DefaultPlugin{
			PluginID:   "testnull",
			PluginType: "null",
		},
		DefaultInputter: pg.DefaultInputter{
			InputChannel: make(pg.EntryChannel, 10),
		},
	}
}
