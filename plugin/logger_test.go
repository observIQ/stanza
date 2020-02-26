package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func NewFakeLoggerPlugin() *LoggerPlugin {
	logger, _ := zap.NewProduction()
	sugaredLogger := logger.Sugar()
	return &LoggerPlugin{
		DefaultPlugin: DefaultPlugin{
			id:         "test",
			pluginType: "logger",
		},
		DefaultInputter: DefaultInputter{
			input: make(EntryChannel, 10),
		},
		SugaredLogger: sugaredLogger,
		logFunc:       func(string, ...interface{}) {},
	}
}

func TestLoggerImplementations(t *testing.T) {
	assert.Implements(t, (*Plugin)(nil), new(LoggerPlugin))
	assert.Implements(t, (*Inputter)(nil), new(LoggerPlugin))
}

func TestLoggerExitsOnInputClose(t *testing.T) {
	logger := NewFakeLoggerPlugin()
	testInputterExitsOnChannelClose(t, logger)
}
