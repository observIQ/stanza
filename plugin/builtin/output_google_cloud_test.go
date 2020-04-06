package builtin

import (
	"testing"

	"cloud.google.com/go/logging"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/helper"
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

func newFakeGoogleCloudLoggingPlugin() *GoogleCloudOutput {
	logger, _ := zap.NewProduction()
	sugaredLogger := logger.Sugar()
	return &GoogleCloudOutput{
		BasicPlugin: helper.BasicPlugin{
			PluginID:      "test",
			PluginType:    "google_cloud_out",
			SugaredLogger: sugaredLogger,
		},
		googleCloudLogger: newFakeGoogleCloudLogger(),
		projectID:         "testproject",
	}
}

func TestGoogleCloudLoggingImplementations(t *testing.T) {
	assert.Implements(t, (*plugin.Plugin)(nil), new(GoogleCloudOutput))
}
