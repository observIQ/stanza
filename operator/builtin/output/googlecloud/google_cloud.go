package googlecloud

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/url"
	"time"

	vkit "cloud.google.com/go/logging/apiv2"
	"github.com/golang/protobuf/ptypes"
	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/errors"
	"github.com/observiq/stanza/internal/version"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/buffer"
	"github.com/observiq/stanza/operator/flusher"
	"github.com/observiq/stanza/operator/helper"
	"go.uber.org/zap"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	mrpb "google.golang.org/genproto/googleapis/api/monitoredres"
	logpb "google.golang.org/genproto/googleapis/logging/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding/gzip"
)

func init() {
	operator.Register("google_cloud_output", func() operator.Builder { return NewGoogleCloudOutputConfig("") })
}

// NewGoogleCloudOutputConfig creates a new google cloud output config with default
func NewGoogleCloudOutputConfig(operatorID string) *GoogleCloudOutputConfig {
	return &GoogleCloudOutputConfig{
		OutputConfig:   helper.NewOutputConfig(operatorID, "google_cloud_output"),
		BufferConfig:   buffer.NewConfig(),
		FlusherConfig:  flusher.NewConfig(),
		Timeout:        helper.Duration{Duration: 30 * time.Second},
		UseCompression: true,
	}
}

// GoogleCloudOutputConfig is the configuration of a google cloud output operator.
type GoogleCloudOutputConfig struct {
	helper.OutputConfig `yaml:",inline"`
	BufferConfig        buffer.Config  `json:"buffer,omitempty" yaml:"buffer,omitempty"`
	FlusherConfig       flusher.Config `json:"flusher,omitempty" yaml:"flusher,omitempty"`

	Credentials     string          `json:"credentials,omitempty"      yaml:"credentials,omitempty"`
	CredentialsFile string          `json:"credentials_file,omitempty" yaml:"credentials_file,omitempty"`
	ProjectID       string          `json:"project_id"                 yaml:"project_id"`
	LogNameField    *entry.Field    `json:"log_name_field,omitempty"   yaml:"log_name_field,omitempty"`
	TraceField      *entry.Field    `json:"trace_field,omitempty"      yaml:"trace_field,omitempty"`
	SpanIDField     *entry.Field    `json:"span_id_field,omitempty"    yaml:"span_id_field,omitempty"`
	Timeout         helper.Duration `json:"timeout,omitempty"          yaml:"timeout,omitempty"`
	UseCompression  bool            `json:"use_compression,omitempty"  yaml:"use_compression,omitempty"`
}

// Build will build a google cloud output operator.
func (c GoogleCloudOutputConfig) Build(buildContext operator.BuildContext) (operator.Operator, error) {
	outputOperator, err := c.OutputConfig.Build(buildContext)
	if err != nil {
		return nil, err
	}

	newBuffer, err := c.BufferConfig.Build(buildContext, c.ID())
	if err != nil {
		return nil, err
	}

	googleCloudOutput := &GoogleCloudOutput{
		OutputOperator:  outputOperator,
		credentials:     c.Credentials,
		credentialsFile: c.CredentialsFile,
		projectID:       c.ProjectID,
		buffer:          newBuffer,
		logNameField:    c.LogNameField,
		traceField:      c.TraceField,
		spanIDField:     c.SpanIDField,
		timeout:         c.Timeout.Raw(),
		useCompression:  c.UseCompression,
	}

	newFlusher := c.FlusherConfig.Build(newBuffer, googleCloudOutput.ProcessMulti, outputOperator.SugaredLogger)
	googleCloudOutput.flusher = newFlusher

	return googleCloudOutput, nil
}

// GoogleCloudOutput is an operator that sends logs to google cloud logging.
type GoogleCloudOutput struct {
	helper.OutputOperator
	buffer  buffer.Buffer
	flusher *flusher.Flusher

	credentials     string
	credentialsFile string
	projectID       string

	logNameField   *entry.Field
	traceField     *entry.Field
	spanIDField    *entry.Field
	useCompression bool

	client  *vkit.Client
	timeout time.Duration
}

// Start will start the google cloud logger.
func (p *GoogleCloudOutput) Start() error {
	var credentials *google.Credentials
	var err error
	scope := "https://www.googleapis.com/auth/logging.write"
	switch {
	case p.credentials != "" && p.credentialsFile != "":
		return errors.NewError("at most one of credentials or credentials_file can be configured", "")
	case p.credentials != "":
		credentials, err = google.CredentialsFromJSON(context.Background(), []byte(p.credentials), scope)
		if err != nil {
			return fmt.Errorf("parse credentials: %s", err)
		}
	case p.credentialsFile != "":
		credentialsBytes, err := ioutil.ReadFile(p.credentialsFile)
		if err != nil {
			return fmt.Errorf("read credentials file: %s", err)
		}
		credentials, err = google.CredentialsFromJSON(context.Background(), credentialsBytes, scope)
		if err != nil {
			return fmt.Errorf("parse credentials: %s", err)
		}
	default:
		credentials, err = google.FindDefaultCredentials(context.Background(), scope)
		if err != nil {
			return fmt.Errorf("get default credentials: %s", err)
		}
	}

	if p.projectID == "" && credentials.ProjectID == "" {
		return fmt.Errorf("no project id found on google creds")
	}

	if p.projectID == "" {
		p.projectID = credentials.ProjectID
	}

	options := make([]option.ClientOption, 0, 2)
	options = append(options, option.WithCredentials(credentials))
	options = append(options, option.WithUserAgent("StanzaLogAgent/"+version.GetVersion()))
	if p.useCompression {
		options = append(options, option.WithGRPCDialOption(grpc.WithDefaultCallOptions(grpc.UseCompressor(gzip.Name))))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := vkit.NewClient(ctx, options...)
	if err != nil {
		return fmt.Errorf("create client: %w", err)
	}
	p.client = client

	// Test writing a log message
	ctx, cancel = context.WithTimeout(context.Background(), p.timeout)
	defer cancel()
	err = p.TestConnection(ctx)
	if err != nil {
		return err
	}

	p.flusher.Start()
	return nil
}

// TestConnection will attempt to send a test entry to google cloud logging
func (p *GoogleCloudOutput) TestConnection(ctx context.Context) error {
	testEntry := entry.New()
	testEntry.Record = map[string]interface{}{"message": "Test connection"}
	if err := p.ProcessMulti(ctx, []*entry.Entry{testEntry}); err != nil {
		return fmt.Errorf("test connection: %s", err)
	}
	return nil
}

// Stop will flush the google cloud logger and close the underlying connection
func (p *GoogleCloudOutput) Stop() error {
	p.flusher.Stop()
	if err := p.buffer.Close(); err != nil {
		return err
	}
	return p.client.Close()
}

func (p *GoogleCloudOutput) Process(ctx context.Context, e *entry.Entry) error {
	return p.buffer.Add(ctx, e)
}

// ProcessMulti will process multiple log entries and send them in batch to google cloud logging.
func (p *GoogleCloudOutput) ProcessMulti(ctx context.Context, entries []*entry.Entry) error {
	pbEntries := make([]*logpb.LogEntry, 0, len(entries))
	for _, entry := range entries {
		pbEntry, err := p.createProtobufEntry(entry)
		if err != nil {
			p.Errorw("Failed to create protobuf entry. Dropping entry", zap.Any("error", err))
			continue
		}
		pbEntries = append(pbEntries, pbEntry)
	}

	req := logpb.WriteLogEntriesRequest{
		LogName:  p.toLogNamePath("default"),
		Entries:  pbEntries,
		Resource: p.defaultResource(),
	}

	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()
	_, err := p.client.WriteLogEntries(ctx, &req)
	if err != nil {
		return fmt.Errorf("write log entries: %s", err)
	}

	return nil
}

func (p *GoogleCloudOutput) defaultResource() *mrpb.MonitoredResource {
	return &mrpb.MonitoredResource{
		Type: "global",
		Labels: map[string]string{
			"project_id": p.projectID,
		},
	}
}

func (p *GoogleCloudOutput) toLogNamePath(logName string) string {
	return fmt.Sprintf("projects/%s/logs/%s", p.projectID, url.PathEscape(logName))
}

func (p *GoogleCloudOutput) createProtobufEntry(e *entry.Entry) (newEntry *logpb.LogEntry, err error) {
	ts, err := ptypes.TimestampProto(e.Timestamp)
	if err != nil {
		return nil, err
	}

	newEntry = &logpb.LogEntry{
		Timestamp: ts,
		Labels:    e.Labels,
	}

	if p.logNameField != nil {
		var rawLogName string
		err := e.Read(*p.logNameField, &rawLogName)
		if err != nil {
			p.Warnw("Failed to set log name", zap.Error(err), "entry", e)
		} else {
			newEntry.LogName = p.toLogNamePath(rawLogName)
			e.Delete(*p.logNameField)
		}
	}

	if p.traceField != nil {
		err := e.Read(*p.traceField, &newEntry.Trace)
		if err != nil {
			p.Warnw("Failed to set trace", zap.Error(err), "entry", e)
		} else {
			e.Delete(*p.traceField)
		}
	}

	if p.spanIDField != nil {
		err := e.Read(*p.spanIDField, &newEntry.SpanId)
		if err != nil {
			p.Warnw("Failed to set span ID", zap.Error(err), "entry", e)
		} else {
			e.Delete(*p.spanIDField)
		}
	}

	newEntry.Severity = convertSeverity(e.Severity)
	err = setPayload(newEntry, e.Record)
	if err != nil {
		return nil, errors.Wrap(err, "set entry payload")
	}

	newEntry.Resource = getResource(e)

	return newEntry, nil
}
