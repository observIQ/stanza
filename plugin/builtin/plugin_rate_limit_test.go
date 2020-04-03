package builtin

import (
	"testing"
	"time"

	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/helper"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func NewFakeRateLimitPlugin() *RateLimitPlugin {
	logger, _ := zap.NewProduction()
	sugaredLogger := logger.Sugar()
	return &RateLimitPlugin{
		BasicIdentity: helper.BasicIdentity{
			PluginID:      "test",
			PluginType:    "rate_filter",
			SugaredLogger: sugaredLogger,
		},
		BasicTransformer: helper.BasicTransformer{
			Output: newFakeNullOutput(),
		},
		interval: time.Millisecond,
		burst:    10,
	}
}

func TestRateLimitImplementations(t *testing.T) {
	assert.Implements(t, (*plugin.Plugin)(nil), new(RateLimitPlugin))
}
