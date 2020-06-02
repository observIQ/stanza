package builtin

import (
	"context"
	"testing"
	"time"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/helper"
	"github.com/stretchr/testify/assert"
)

func NewFakeCopyPlugin() *CopyPlugin {
	out1 := newFakeNullOutput()
	out1.PluginID = "out1"

	out2 := newFakeNullOutput()
	out2.PluginID = "out2"
	return &CopyPlugin{
		BasicPlugin: helper.BasicPlugin{
			PluginID:   "test",
			PluginType: "copy",
		},
		outputs: []plugin.Plugin{
			out1,
			out2,
		},
	}
}

func TestCopyImplementations(t *testing.T) {
	assert.Implements(t, (*plugin.Plugin)(nil), new(CopyPlugin))
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
		err := copy.Process(context.Background(), &entry.Entry{
			Timestamp: time.Now(),
			Record:    record,
		})

		if err != nil {
			b.FailNow()
		}
	}
}
