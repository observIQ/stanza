package builtin

import (
	"testing"
	"time"

	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/base"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func NewFakeRateLimitPlugin() *RateFilter {
	logger, _ := zap.NewProduction()
	sugaredLogger := logger.Sugar()
	return &RateFilter{
		FilterPlugin: base.FilterPlugin{
			InputPlugin: base.InputPlugin{
				Plugin: base.Plugin{
					PluginID:      "test",
					PluginType:    "rate_filter",
					SugaredLogger: sugaredLogger,
				},
				Output: newFakeNullOutput(),
			},
		},
		Interval: time.Millisecond,
		Burst:    10,
	}
}

func TestRateLimitImplementations(t *testing.T) {
	assert.Implements(t, (*plugin.Plugin)(nil), new(RateFilter))
	assert.Implements(t, (*plugin.Producer)(nil), new(RateFilter))
	assert.Implements(t, (*plugin.Consumer)(nil), new(RateFilter))
}
