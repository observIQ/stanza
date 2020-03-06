package plugins

import (
	"testing"
	"time"

	pg "github.com/bluemedora/bplogagent/plugin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
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
		DefaultInputter: pg.DefaultInputter{
			InputChannel: make(pg.EntryChannel, 10),
		},
		DefaultOutputter: pg.DefaultOutputter{
			OutputPlugin: newFakeNullOutput(),
		},
		interval: time.Millisecond,
		burst:    10,
	}
}

func TestRateLimitImplementations(t *testing.T) {
	assert.Implements(t, (*pg.Outputter)(nil), new(RateLimitPlugin))
	assert.Implements(t, (*pg.Inputter)(nil), new(RateLimitPlugin))
	assert.Implements(t, (*pg.Plugin)(nil), new(RateLimitPlugin))
}

func TestRateLimitExitsOnInputClose(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreTopFunction("go.opencensus.io/stats/view.(*worker).start"))
	rateLimit := NewFakeRateLimitPlugin()
	testInputterExitsOnChannelClose(t, rateLimit)
}
