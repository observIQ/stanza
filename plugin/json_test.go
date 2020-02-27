package plugin

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
	"go.uber.org/zap"
)

func NewFakeJSONPlugin() *JSONPlugin {
	logger, _ := zap.NewProduction()
	return &JSONPlugin{
		DefaultPlugin: DefaultPlugin{
			id:         "test",
			pluginType: "json",
		},
		DefaultInputter: DefaultInputter{
			input: make(EntryChannel, 10),
		},
		DefaultOutputter: DefaultOutputter{
			output:         make(EntryChannel, 10),
			outputPluginID: "testoutput",
		},
		SugaredLogger:    logger.Sugar(),
		field:            "testfield",
		destinationField: "testparsed",
	}
}

func TestJSONImplementations(t *testing.T) {
	assert.Implements(t, (*Outputter)(nil), new(JSONPlugin))
	assert.Implements(t, (*Inputter)(nil), new(JSONPlugin))
	assert.Implements(t, (*Plugin)(nil), new(JSONPlugin))
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
