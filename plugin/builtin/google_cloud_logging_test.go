package builtin

import (
	"testing"

	"cloud.google.com/go/logging"
	pg "github.com/bluemedora/bplogagent/plugin"
	"github.com/stretchr/testify/assert"
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
		googleCloudLogger: newFakeGoogleCloudLogger(),
		projectID:         "testproject",
	}
}

func TestGoogleCloudLoggingImplementations(t *testing.T) {
	assert.Implements(t, (*pg.Inputter)(nil), new(GoogleCloudLoggingPlugin))
	assert.Implements(t, (*pg.Plugin)(nil), new(GoogleCloudLoggingPlugin))
	assert.Implements(t, (*pg.Starter)(nil), new(GoogleCloudLoggingPlugin))
	assert.Implements(t, (*pg.Stopper)(nil), new(GoogleCloudLoggingPlugin))
}
