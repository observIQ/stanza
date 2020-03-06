package plugins

import (
	"testing"

	pg "github.com/bluemedora/bplogagent/plugin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
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
		DefaultInputter: pg.DefaultInputter{
			InputChannel: make(pg.EntryChannel, 10),
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

func TestCopyExitsOnChannelClose(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreTopFunction("go.opencensus.io/stats/view.(*worker).start"))
	copy := NewFakeCopyPlugin()
	testInputterExitsOnChannelClose(t, copy)
}

func BenchmarkCopy(b *testing.B) {
	for _, bm := range standardInputterBenchmarks {
		b.Run(bm.String(), func(b *testing.B) {
			copyPlugin := NewFakeCopyPlugin()
			benchmarkInputter(b, copyPlugin, bm, generateRandomNestedMap)
		})
	}
}
