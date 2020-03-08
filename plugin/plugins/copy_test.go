package plugins

import (
	"testing"

	pg "github.com/bluemedora/bplogagent/plugin"
	"github.com/stretchr/testify/assert"
)

func NewFakeCopyPlugin() *CopyPlugin {
	out1 := newFakeNullOutput()
	out1.PluginID = "out1"

	out2 := newFakeNullOutput()
	out2.PluginID = "out2"
	return &CopyPlugin{
		DefaultPlugin: pg.DefaultPlugin{
			PluginID:   "test",
			PluginType: "copy",
		},
		outputs: []pg.Inputter{
			out1,
			out2,
		},
	}
}

func TestCopyImplementations(t *testing.T) {
	assert.Implements(t, (*pg.Outputter)(nil), new(CopyPlugin))
	assert.Implements(t, (*pg.Inputter)(nil), new(CopyPlugin))
	assert.Implements(t, (*pg.Plugin)(nil), new(CopyPlugin))
}
