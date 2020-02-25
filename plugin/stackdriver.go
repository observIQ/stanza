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
	RegisterConfig("stackdriver", &StackdriverOutputConfig{})
}

type StackdriverOutputConfig struct {
	DefaultDestinationConfig `mapstructure:",squash"`
	Credentials              string
	ProjectID                string `mapstructure:"project_id"`
}

func (c *StackdriverOutputConfig) Build(logger *zap.SugaredLogger) (Plugin, error) {
	options := make([]option.ClientOption, 0, 2)

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

	stackdriverLogger := client.Logger("test_log_name", logging.ConcurrentWriteLimit(10))

	dest := &StackdriverPlugin{
		DefaultDestination: c.DefaultDestinationConfig.Build(),
		logger:             stackdriverLogger,
		ProjectID:          c.ProjectID,
		SugaredLogger:      logger,
	}

	return dest, nil
}

type StackdriverPlugin struct {
	DefaultDestination
	logger    *logging.Logger
	ProjectID string
	*zap.SugaredLogger
}

func (p *StackdriverPlugin) Start(wg *sync.WaitGroup) error {
	go func() {
		defer wg.Done()
		defer p.logger.Flush()

		for {
			entry, ok := <-p.DefaultDestination.Input()
			if !ok {
				return
			}

			stackdriverEntry := logging.Entry{
				Timestamp: entry.Timestamp,
				Payload:   entry.Record,
				Severity:  logging.Info, // TODO calculate severity correctly
			}

			p.logger.Log(stackdriverEntry)
		}
	}()

	return nil
}
