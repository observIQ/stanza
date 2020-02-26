package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func NewFakeCopyPlugin() *CopyPlugin {
	return &CopyPlugin{
		DefaultPlugin: DefaultPlugin{
			id:         "test",
			pluginType: "copy",
		},
		DefaultInputter: DefaultInputter{
			input: make(EntryChannel, 10),
		},
		outputs: map[PluginID]EntryChannel{
			"out1": make(EntryChannel, 10),
			"out2": make(EntryChannel, 10),
		},
		outputIDs: []PluginID{
			"out1",
			"out2",
		},
	}
}

func TestCopyImplementations(t *testing.T) {
	assert.Implements(t, (*Outputter)(nil), new(CopyPlugin))
	assert.Implements(t, (*Inputter)(nil), new(CopyPlugin))
	assert.Implements(t, (*Plugin)(nil), new(CopyPlugin))
}

func TestCopyExitsOnChannelClose(t *testing.T) {
	copy := NewFakeCopyPlugin()
	testInputterExitsOnChannelClose(t, copy)
}
