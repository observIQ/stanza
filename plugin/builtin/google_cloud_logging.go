package builtin

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/logging"
	"github.com/bluemedora/bplogagent/entry"
	pg "github.com/bluemedora/bplogagent/plugin"
	"google.golang.org/api/option"
)

func init() {
	pg.RegisterConfig("google_cloud_logging", &GoogleCloudLoggingOutputConfig{})
}

type GoogleCloudLoggingOutputConfig struct {
	pg.DefaultPluginConfig `mapstructure:",squash" yaml:",inline"`
	Credentials            string
	ProjectID              string `mapstructure:"project_id"`
}

func (c GoogleCloudLoggingOutputConfig) Build(buildContext pg.BuildContext) (pg.Plugin, error) {

	// TODO configure bundle size
	// TODO allow alternate credentials options (file, etc.)
	if c.Credentials == "" {
		return nil, errors.New("missing required configuration option credentials")
	}

	if c.ProjectID == "" {
		return nil, errors.New("missing required configuration option project_id")
	}

	defaultPlugin, err := c.DefaultPluginConfig.Build(buildContext.Logger)
	if err != nil {
		return nil, fmt.Errorf("build default plugin: %s", err)
	}

	dest := &GoogleCloudLoggingPlugin{
		DefaultPlugin: defaultPlugin,

		credentials: c.Credentials,
		projectID:   c.ProjectID,
	}

	return dest, nil
}

type GoogleCloudLogger interface {
	Log(logging.Entry)
	Flush() error
}

type GoogleCloudLoggingPlugin struct {
	pg.DefaultPlugin

	credentials       string
	projectID         string
	googleCloudLogger GoogleCloudLogger
}

func (p *GoogleCloudLoggingPlugin) Start() error {
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

func (p *GoogleCloudLoggingPlugin) Stop() {
	err := p.googleCloudLogger.Flush()
	if err != nil {
		panic(err)
	}
}

func (p *GoogleCloudLoggingPlugin) Input(entry *entry.Entry) error {
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
