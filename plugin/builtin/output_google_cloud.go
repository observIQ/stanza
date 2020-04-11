package builtin

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/logging"
	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/helper"
	"go.uber.org/zap"
	"google.golang.org/api/option"
)

func init() {
	plugin.Register("google_cloud_output", &GoogleCloudOutputConfig{})
}

// GoogleCloudOutputConfig is the configuration of a google cloud output plugin.
type GoogleCloudOutputConfig struct {
	helper.BasicPluginConfig `mapstructure:",squash" yaml:",inline"`
	Credentials              string              `mapstructure:"credentials"    yaml:"credentials"`
	ProjectID                string              `mapstructure:"project_id"     yaml:"credentials"`
	LogNameField             entry.FieldSelector `mapstructure:"log_name_field" yaml:"log_name_field"`
	LabelsField              entry.FieldSelector `mapstructure:"labels_field"   yaml:"labels_field"`
	SeverityField            entry.FieldSelector `mapstructure:"severity_field" yaml:"severity_field"`
	TraceField               entry.FieldSelector `mapstructure:"trace_field"    yaml:"trace_field"`
	SpanIDField              entry.FieldSelector `mapstructure:"span_id_field"  yaml:"span_id_field"`
}

// Build will build a google cloud output plugin.
func (c GoogleCloudOutputConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	basicPlugin, err := c.BasicPluginConfig.Build(context.Logger)
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
		BasicPlugin: basicPlugin,
		credentials: c.Credentials,
		projectID:   c.ProjectID,

		logNameField:  c.LogNameField,
		labelsField:   c.LabelsField,
		severityField: c.SeverityField,
		traceField:    c.TraceField,
		spanIDField:   c.SpanIDField,
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
	helper.BasicPlugin
	helper.BasicOutput

	credentials       string
	projectID         string
	googleCloudLogger GoogleCloudLogger

	logNameField  entry.FieldSelector
	labelsField   entry.FieldSelector
	severityField entry.FieldSelector
	traceField    entry.FieldSelector
	spanIDField   entry.FieldSelector
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

// Process will send an entry to google cloud logging.
func (p *GoogleCloudOutput) Process(entry *entry.Entry) error {
	cloudEntry := logging.Entry{
		Timestamp: entry.Timestamp,
		Payload:   entry.Record,
		Severity:  logging.Info, // TODO calculate severity correctly
	}

	// TODO when using the logger.Write function, this doesn't work.
	// We'll need to use the WriteLogEntries call directly I think

	// if p.logNameField != nil {
	// 	err := entry.Read(p.logNameField, &cloudEntry.LogName)
	// 	if err != nil {
	// 		p.Warnw("Failed to set log name", zap.Error(err))
	// 	} else {
	// 		entry.Delete(p.logNameField)
	// 	}
	// }

	if p.labelsField != nil {
		err := entry.Read(p.labelsField, &cloudEntry.Labels)
		if err != nil {
			p.Warnw("Failed to set labels", zap.Error(err))
		} else {
			entry.Delete(p.labelsField)
		}
	}

	if p.traceField != nil {
		err := entry.Read(p.traceField, &cloudEntry.Trace)
		if err != nil {
			p.Warnw("Failed to set trace", zap.Error(err))
		} else {
			entry.Delete(p.traceField)
		}
	}

	if p.spanIDField != nil {
		err := entry.Read(p.spanIDField, &cloudEntry.SpanID)
		if err != nil {
			p.Warnw("Failed to set span ID", zap.Error(err))
		} else {
			entry.Delete(p.spanIDField)
		}
	}

	if p.severityField != nil {
		var severityString string
		err := entry.Read(p.severityField, &severityString)
		if err != nil {
			p.Warnw("Failed to set severity", zap.Error(err))
		} else {
			entry.Delete(p.severityField)
		}
		cloudEntry.Severity = logging.ParseSeverity(severityString)
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
	p.googleCloudLogger.Log(cloudEntry)

	return nil
}
