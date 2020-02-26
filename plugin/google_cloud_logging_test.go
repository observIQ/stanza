package plugin

import (
	"testing"

	"cloud.google.com/go/logging"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
	"go.uber.org/zap"
)

type FakeStackdriverLogger struct {
	log   func(logging.Entry)
	flush func() error
}

func (l FakeStackdriverLogger) Log(entry logging.Entry) {
	l.log(entry)
}

func (l FakeStackdriverLogger) Flush() error {
	return l.flush()
}

func newFakeStackdriverLogger() StackdriverLogger {
	return &FakeStackdriverLogger{
		log:   func(logging.Entry) {},
		flush: func() error { return nil },
	}
}

func newFakeStackdriverPlugin() *StackdriverPlugin {
	logger, _ := zap.NewProduction()
	sugaredLogger := logger.Sugar()
	return &StackdriverPlugin{
		DefaultPlugin: DefaultPlugin{
			id:         "test",
			pluginType: "stackdriver",
		},
		DefaultInputter: DefaultInputter{
			input: make(EntryChannel, 10),
		},
		stackdriverLogger: newFakeStackdriverLogger(),
		ProjectID:         "testproject",
		SugaredLogger:     sugaredLogger,
	}
}

func TestStackdriverImplementations(t *testing.T) {
	assert.Implements(t, (*Inputter)(nil), new(StackdriverPlugin))
	assert.Implements(t, (*Plugin)(nil), new(StackdriverPlugin))
}

func TestStackdriverExitsOnInputClose(t *testing.T) {
	// TODO Remove ignore when this is fixed https://github.com/census-instrumentation/opencensus-go/issues/1191
	defer goleak.VerifyNone(t, goleak.IgnoreTopFunction("go.opencensus.io/stats/view.(*worker).start"))
	stackdriver := newFakeStackdriverPlugin()
	testInputterExitsOnChannelClose(t, stackdriver)
}
