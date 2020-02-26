package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
		SugaredLogger: logger.Sugar(),
		field:         "testfield",
	}
}

func TestJSONImplementations(t *testing.T) {
	assert.Implements(t, (*Outputter)(nil), new(JSONPlugin))
	assert.Implements(t, (*Inputter)(nil), new(JSONPlugin))
	assert.Implements(t, (*Plugin)(nil), new(JSONPlugin))
}

func TestJSONExitsOnInputClose(t *testing.T) {
	json := NewFakeJSONPlugin()
	testInputterExitsOnChannelClose(t, json)
}
