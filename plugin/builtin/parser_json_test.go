package builtin

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/base"
	jsoniter "github.com/json-iterator/go"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func NewFakeJSONPlugin() *JSONParser {
	logger, _ := zap.NewProduction()
	return &JSONParser{
		ParserPlugin: base.ParserPlugin{
			Plugin: base.Plugin{
				PluginID:      "test",
				PluginType:    "json",
				SugaredLogger: logger.Sugar(),
			},
			Output: nil,
		},
		field:            "testfield",
		destinationField: "testparsed",
		json:             jsoniter.ConfigFastest,
	}
}

func TestJSONImplementations(t *testing.T) {
	assert.Implements(t, (*plugin.Plugin)(nil), new(JSONParser))
	assert.Implements(t, (*plugin.Consumer)(nil), new(JSONParser))
}

func BenchmarkJSONParser(b *testing.B) {
	for _, ib := range standardInputterBenchmarks {
		ib := ib
		b.Run(ib.String(), func(b *testing.B) {
			benchJSONParser(b, ib)
		})
	}
}

func benchJSONParser(b *testing.B, ib inputterBenchmark) {
	copy := NewFakeJSONPlugin()
	record := generateRandomNestedMap(ib.fields, ib.depth, ib.fieldLength)
	marshalled, err := json.Marshal(record)
	assert.NoError(b, err)
	marshalledRecord := map[string]interface{}{
		"testfield": string(marshalled),
	}

	b.SetBytes(ib.EstimatedBytes())
	for i := 0; i < b.N; i++ {
		err := copy.Consume(&entry.Entry{
			Timestamp: time.Now(),
			Record:    marshalledRecord,
		})

		if err != nil {
			b.FailNow()
		}
	}
}
