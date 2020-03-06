package plugins

import (
	"testing"

	"cloud.google.com/go/logging"
	pg "github.com/bluemedora/bplogagent/plugin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
	"go.uber.org/zap"
)

type FakeGoogleCloudLogger struct {
	log   func(logging.Entry)
	flush func() error
}

func (l FakeGoogleCloudLogger) Log(entry logging.Entry) {
	l.log(entry)
}

func (l FakeGoogleCloudLogger) Flush() error {
	return l.flush()
}

func newFakeGoogleCloudLogger() GoogleCloudLogger {
	return &FakeGoogleCloudLogger{
		log:   func(logging.Entry) {},
		flush: func() error { return nil },
	}
}

func newFakeGoogleCloudLoggingPlugin() *GoogleCloudLoggingPlugin {
	logger, _ := zap.NewProduction()
	sugaredLogger := logger.Sugar()
	return &GoogleCloudLoggingPlugin{
		DefaultPlugin: pg.DefaultPlugin{
			PluginID:      "test",
			PluginType:    "GoogleCloudLogging",
			SugaredLogger: sugaredLogger,
		},
		DefaultInputter: pg.DefaultInputter{
			InputChannel: make(pg.EntryChannel, 10),
		},
		googleCloudLogger: newFakeGoogleCloudLogger(),
		projectID:         "testproject",
	}
}

func TestGoogleCloudLoggingImplementations(t *testing.T) {
	assert.Implements(t, (*pg.Inputter)(nil), new(GoogleCloudLoggingPlugin))
	assert.Implements(t, (*pg.Plugin)(nil), new(GoogleCloudLoggingPlugin))
}

func TestGoogleCloudLoggingExitsOnInputClose(t *testing.T) {
	// TODO Remove ignore when this is fixed https://github.com/census-instrumentation/opencensus-go/issues/1191
	defer goleak.VerifyNone(t, goleak.IgnoreTopFunction("go.opencensus.io/stats/view.(*worker).start"))
	googleCloudLogging := newFakeGoogleCloudLoggingPlugin()
	testInputterExitsOnChannelClose(t, googleCloudLogging)
}
