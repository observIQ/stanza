package plugins

import (
	"testing"
	"time"

	pg "github.com/bluemedora/bplogagent/plugin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func NewFakeRateLimitPlugin() *RateLimitPlugin {
	logger, _ := zap.NewProduction()
	sugaredLogger := logger.Sugar()
	return &RateLimitPlugin{
		DefaultPlugin: pg.DefaultPlugin{
			PluginID:      "test",
			PluginType:    "rate_limit",
			SugaredLogger: sugaredLogger,
		},
		DefaultOutputter: pg.DefaultOutputter{
			OutputPlugin: newFakeNullOutput(),
		},
		Interval: time.Millisecond,
		Burst:    10,
	}
}

func TestRateLimitImplementations(t *testing.T) {
	assert.Implements(t, (*pg.Outputter)(nil), new(RateLimitPlugin))
	assert.Implements(t, (*pg.Inputter)(nil), new(RateLimitPlugin))
	assert.Implements(t, (*pg.Plugin)(nil), new(RateLimitPlugin))
}
