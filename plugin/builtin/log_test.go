package builtin

import (
	"testing"

	pg "github.com/bluemedora/bplogagent/plugin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func NewFakeLogOutput() *LogOutput {
	logger, _ := zap.NewProduction()
	sugaredLogger := logger.Sugar()
	return &LogOutput{
		DefaultPlugin: pg.DefaultPlugin{
			PluginID:      "test",
			PluginType:    "logger",
			SugaredLogger: sugaredLogger,
		},
		logFunc: func(string, ...interface{}) {},
	}
}

func TestLoggerImplementations(t *testing.T) {
	assert.Implements(t, (*pg.Plugin)(nil), new(LogOutput))
	assert.Implements(t, (*pg.Inputter)(nil), new(LogOutput))
}
