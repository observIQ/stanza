package plugin

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
	"go.uber.org/zap"
)

func NewFakeRateLimitPlugin() *RateLimitPlugin {
	logger, _ := zap.NewProduction()
	sugaredLogger := logger.Sugar()
	return &RateLimitPlugin{
		DefaultPlugin: DefaultPlugin{
			id:            "test",
			pluginType:    "rate_limit",
			SugaredLogger: sugaredLogger,
		},
		DefaultInputter: DefaultInputter{
			input: make(EntryChannel, 10),
		},
		DefaultOutputter: DefaultOutputter{
			outputPlugin: newFakeNullOutput(),
		},
		interval: time.Millisecond,
		burst:    10,
	}
}

func TestRateLimitImplementations(t *testing.T) {
	assert.Implements(t, (*Outputter)(nil), new(RateLimitPlugin))
	assert.Implements(t, (*Inputter)(nil), new(RateLimitPlugin))
	assert.Implements(t, (*Plugin)(nil), new(RateLimitPlugin))
}

func TestRateLimitExitsOnInputClose(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreTopFunction("go.opencensus.io/stats/view.(*worker).start"))
	rateLimit := NewFakeRateLimitPlugin()
	testInputterExitsOnChannelClose(t, rateLimit)
}
