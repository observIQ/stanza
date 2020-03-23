package builtin

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/bluemedora/bplogagent/entry"
	pg "github.com/bluemedora/bplogagent/plugin"

	"github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func NewFakeJSONPlugin() *JSONParser {
	logger, _ := zap.NewProduction()
	return &JSONParser{
		DefaultPlugin: pg.DefaultPlugin{
			PluginID:      "test",
			PluginType:    "json",
			SugaredLogger: logger.Sugar(),
		},
		DefaultOutputter: pg.DefaultOutputter{
			OutputPlugin: newFakeNullOutput(),
		},
		field:            "testfield",
		destinationField: "testparsed",
		json:             jsoniter.ConfigFastest,
	}
}

func TestJSONImplementations(t *testing.T) {
	assert.Implements(t, (*pg.Outputter)(nil), new(JSONParser))
	assert.Implements(t, (*pg.Inputter)(nil), new(JSONParser))
	assert.Implements(t, (*pg.Plugin)(nil), new(JSONParser))
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
		err := copy.Input(&entry.Entry{
			Timestamp: time.Now(),
			Record:    marshalledRecord,
		})

		if err != nil {
			b.FailNow()
		}
	}
}
