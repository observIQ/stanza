package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

func NewFakeCopyPlugin() *CopyPlugin {
	out1 := newFakeNullOutput()
	out1.id = "out1"

	out2 := newFakeNullOutput()
	out2.id = "out2"
	return &CopyPlugin{
		DefaultPlugin: DefaultPlugin{
			id:         "test",
			pluginType: "copy",
		},
		DefaultInputter: DefaultInputter{
			input: make(EntryChannel, 10),
		},
		outputs: []Inputter{
			out1,
			out2,
		},
	}
}

func TestCopyImplementations(t *testing.T) {
	assert.Implements(t, (*Outputter)(nil), new(CopyPlugin))
	assert.Implements(t, (*Inputter)(nil), new(CopyPlugin))
	assert.Implements(t, (*Plugin)(nil), new(CopyPlugin))
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
