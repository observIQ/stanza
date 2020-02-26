package plugin

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func NewFakeRateLimitPlugin() *RateLimitPlugin {
	logger, _ := zap.NewProduction()
	sugaredLogger := logger.Sugar()
	return &RateLimitPlugin{
		DefaultPlugin: DefaultPlugin{
			id:         "test",
			pluginType: "rate_limit",
		},
		DefaultInputter: DefaultInputter{
			input: make(EntryChannel, 10),
		},
		DefaultOutputter: DefaultOutputter{
			output:         make(EntryChannel, 10),
			outputPluginID: "testoutput",
		},
		SugaredLogger: sugaredLogger,
		interval:      time.Millisecond,
		burst:         10,
	}
}

func TestRateLimitImplementations(t *testing.T) {
	assert.Implements(t, (*Outputter)(nil), new(RateLimitPlugin))
	assert.Implements(t, (*Inputter)(nil), new(RateLimitPlugin))
	assert.Implements(t, (*Plugin)(nil), new(RateLimitPlugin))
}

func TestRateLimitExitsOnInputClose(t *testing.T) {
	rateLimit := NewFakeRateLimitPlugin()
	testInputterExitsOnChannelClose(t, rateLimit)
}
