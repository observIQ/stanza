package builtin

import (
	"testing"
	"time"

	"github.com/bluemedora/bplogagent/entry"
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

func BenchmarkCopyPlugin(b *testing.B) {
	for _, ib := range standardInputterBenchmarks {
		ib := ib
		b.Run(ib.String(), func(b *testing.B) {
			benchCopyPlugin(b, ib)
		})
	}
}

func benchCopyPlugin(b *testing.B, ib inputterBenchmark) {
	copy := NewFakeCopyPlugin()
	record := generateRandomNestedMap(ib.fields, ib.depth, ib.fieldLength)

	b.SetBytes(ib.EstimatedBytes())
	for i := 0; i < b.N; i++ {
		err := copy.Input(&entry.Entry{
			Timestamp: time.Now(),
			Record:    record,
		})

		if err != nil {
			b.FailNow()
		}
	}
}
