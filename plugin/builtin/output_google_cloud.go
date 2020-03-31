package builtin

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/logging"
	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/base"
	"google.golang.org/api/option"
)

func init() {
	plugin.Register("google_cloud_output", &GoogleCloudOutputConfig{})
}

// GoogleCloudOutputConfig is the configuration of a google cloud output plugin.
type GoogleCloudOutputConfig struct {
	base.OutputConfig `mapstructure:",squash" yaml:",inline"`
	Credentials       string
	ProjectID         string `mapstructure:"project_id"`
}

// Build will build a google cloud output plugin.
func (c GoogleCloudOutputConfig) Build(buildContext plugin.BuildContext) (plugin.Plugin, error) {
	outputPlugin, err := c.OutputConfig.Build(buildContext)
	if err != nil {
		return nil, err
	}

	// TODO configure bundle size
	// TODO allow alternate credentials options (file, etc.)
	if c.Credentials == "" {
		return nil, errors.New("missing required configuration option credentials")
	}

	if c.ProjectID == "" {
		return nil, errors.New("missing required configuration option project_id")
	}

	googleCloudOutput := &GoogleCloudOutput{
		OutputPlugin: outputPlugin,
		credentials:  c.Credentials,
		projectID:    c.ProjectID,
	}

	return googleCloudOutput, nil
}

// GoogleCloudLogger is a logger that logs to google cloud.
type GoogleCloudLogger interface {
	Log(logging.Entry)
	Flush() error
}

// GoogleCloudOutput is a plugin that sends logs to google cloud logging.
type GoogleCloudOutput struct {
	base.OutputPlugin

	credentials       string
	projectID         string
	googleCloudLogger GoogleCloudLogger
}

// Start will start the google cloud logger.
func (p *GoogleCloudOutput) Start() error {
	options := make([]option.ClientOption, 0, 2)
	options = append(options, option.WithCredentialsJSON([]byte(p.credentials)))
	options = append(options, option.WithUserAgent("BindplaneLogAgent/2.0.0"))
	// TODO WithCompressor is deprecated, and may be removed in favor of UseCompressor
	// However, I can't seem to get UseCompressor to work, so skipping for now
	// This seems to be causing flush to hang.
	// options = append(options, option.WithGRPCDialOption(grpc.WithCompressor(grpc.NewGZIPCompressor())))
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*10))
	defer cancel()
	client, err := logging.NewClient(ctx, p.projectID, options...)
	if err != nil {
		return fmt.Errorf("create logging client: %w", err)
	}
	// TODO client.Ping(). Maybe should be in the Start() method

	GoogleCloudLoggingLogger := client.Logger("test_log_name", logging.ConcurrentWriteLimit(10))

	p.googleCloudLogger = GoogleCloudLoggingLogger

	return nil
}

// Stop will flush the google cloud logger.
func (p *GoogleCloudOutput) Stop() error {
	return p.googleCloudLogger.Flush()
}

// Consume will send an entry to google cloud logging.
func (p *GoogleCloudOutput) Consume(entry *entry.Entry) error {
	googleCloudLoggingEntry := logging.Entry{
		Timestamp: entry.Timestamp,
		Payload:   entry.Record,
		Severity:  logging.Info, // TODO calculate severity correctly
	}

	// TODO how do we communicate which logs have been flushed?
	// It appears that there is no way to inject any sort of callback
	// or synchronously log multiple at a time with the current API.
	//
	// To be guarantee delivery, we either need to periodically flush,
	// or request a change to the library. Realistically, a periodic
	// flush is probably pretty practical.
	//
	// Ideas for a library change:
	// - Add a callback to each log entry
	// - Create a Logger.LogMultipleSync() and do our own bundling
	p.googleCloudLogger.Log(googleCloudLoggingEntry)

	return nil
}
