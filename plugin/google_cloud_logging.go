package plugin

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"cloud.google.com/go/logging"
	"go.uber.org/zap"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
)

func init() {
	RegisterConfig("google_cloud_logging", &GoogleCloudLoggingOutputConfig{})
}

type GoogleCloudLoggingOutputConfig struct {
	DefaultPluginConfig   `mapstructure:",squash"`
	DefaultInputterConfig `mapstructure:",squash"`
	Credentials           string
	ProjectID             string `mapstructure:"project_id"`
}

func (c GoogleCloudLoggingOutputConfig) Build(plugins map[PluginID]Plugin, logger *zap.SugaredLogger) (Plugin, error) {
	options := make([]option.ClientOption, 0, 2)

	// TODO configure bundle size
	// TODO allow alternate credentials options (file, etc.)
	if c.Credentials == "" {
		return nil, errors.New("missing required configuration option credentials")
	}

	options = append(options, option.WithCredentialsJSON([]byte(c.Credentials)))
	options = append(options, option.WithUserAgent("BindplaneLogAgent/2.0.0"))
	// TODO WithCompressor is deprecated, and may be removed in favor of UseCompressor
	// However, I can't seem to get UseCompressor to work, so skipping for now
	options = append(options, option.WithGRPCDialOption(grpc.WithCompressor(grpc.NewGZIPCompressor())))

	if c.ProjectID == "" {
		return nil, errors.New("missing required configuration option project_id")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*10))
	defer cancel()
	client, err := logging.NewClient(ctx, c.ProjectID, options...)
	if err != nil {
		return nil, fmt.Errorf("failed to create logging client: %w", err)
	}
	// TODO client.Ping(). Maybe should be in the Start() method

	GoogleCloudLoggingLogger := client.Logger("test_log_name", logging.ConcurrentWriteLimit(10))

	defaultPlugin, err := c.DefaultPluginConfig.Build(logger)
	if err != nil {
		return nil, fmt.Errorf("failed to build default plugin: %s", err)
	}

	defaultInputter, err := c.DefaultInputterConfig.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build default inputter: %s", err)
	}

	dest := &GoogleCloudLoggingPlugin{
		DefaultPlugin:   defaultPlugin,
		DefaultInputter: defaultInputter,

		googleCloudLogger: GoogleCloudLoggingLogger,
		projectID:         c.ProjectID,
	}

	return dest, nil
}

type GoogleCloudLogger interface {
	Log(logging.Entry)
	Flush() error
}

type GoogleCloudLoggingPlugin struct {
	DefaultPlugin
	DefaultInputter

	googleCloudLogger GoogleCloudLogger
	projectID         string
}

func (p *GoogleCloudLoggingPlugin) Start(wg *sync.WaitGroup) error {
	go func() {
		defer wg.Done()
		defer func() {
			p.Infow("Flushing")
			// TODO figure out why this seems to randomly block forever
			err := p.googleCloudLogger.Flush()
			if err != nil {
				p.Errorw("Failed to flush to stackdriver", "error", err)
			}
			p.Infow("Flushed")
		}()

		for {
			entry, ok := <-p.Input()
			if !ok {
				return
			}

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
		}
	}()

	return nil
}
