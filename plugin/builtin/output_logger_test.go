package builtin

import (
	"testing"

	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/base"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func NewFakeLogOutput() *LoggerOutput {
	logger, _ := zap.NewProduction()
	sugaredLogger := logger.Sugar()
	return &LoggerOutput{
		OutputPlugin: base.OutputPlugin{
			base.Plugin{
				PluginID:      "test",
				PluginType:    "logger_output",
				SugaredLogger: sugaredLogger,
			},
		},
		logFunc: func(string, ...interface{}) {},
	}
}

func TestLoggerImplementations(t *testing.T) {
	assert.Implements(t, (*plugin.Plugin)(nil), new(LoggerOutput))
	assert.Implements(t, (*plugin.Consumer)(nil), new(LoggerOutput))
}
