package plugins

import (
	"testing"

	pg "github.com/bluemedora/bplogagent/plugin"

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
	}
}

func TestJSONImplementations(t *testing.T) {
	assert.Implements(t, (*pg.Outputter)(nil), new(JSONParser))
	assert.Implements(t, (*pg.Inputter)(nil), new(JSONParser))
	assert.Implements(t, (*pg.Plugin)(nil), new(JSONParser))
}
