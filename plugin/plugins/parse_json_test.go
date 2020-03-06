package plugins

import (
	"encoding/json"
	"testing"

	pg "github.com/bluemedora/bplogagent/plugin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
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
		DefaultInputter: pg.DefaultInputter{
			InputChannel: make(pg.EntryChannel, 10),
		},
		DefaultOutputter: pg.DefaultOutputter{
			OutputPlugin: newFakeNullOutput(),
		},
		field:            "testfield",
		destinationField: "testparsed",
	}
}

func TestJSONImplementations(t *testing.T) {
	assert.Implements(t, (*pg.Outputter)(nil), new(JSONParser))
	assert.Implements(t, (*pg.Inputter)(nil), new(JSONParser))
	assert.Implements(t, (*pg.Plugin)(nil), new(JSONParser))
}

func TestJSONExitsOnInputClose(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreTopFunction("go.opencensus.io/stats/view.(*worker).start"))
	json := NewFakeJSONPlugin()
	testInputterExitsOnChannelClose(t, json)
}

// TODO write benchmarks for other plugins
func BenchmarkJSON(b *testing.B) {
	for _, bm := range standardInputterBenchmarks {
		b.Run(bm.String(), func(b *testing.B) {
			copyPlugin := NewFakeJSONPlugin()
			benchmarkInputter(b, copyPlugin, bm, generateRandomTestfield)
		})
	}
}

func generateRandomTestfield(fields, depth, length int) map[string]interface{} {
	marshalled, _ := json.Marshal(generateRandomNestedMap(fields, depth, length))
	return map[string]interface{}{
		"testfield": string(marshalled),
	}
}
