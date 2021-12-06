package googlecloud

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/url"
	"strings"
	"sync"
	"time"

	vkit "cloud.google.com/go/logging/apiv2"
	"github.com/golang/protobuf/ptypes"
	"github.com/observiq/stanza/v2/operator/buffer"
	"github.com/observiq/stanza/v2/operator/flusher"
	"github.com/observiq/stanza/v2/version"
	"github.com/open-telemetry/opentelemetry-log-collection/entry"
	otelerrors "github.com/open-telemetry/opentelemetry-log-collection/errors"
	"github.com/open-telemetry/opentelemetry-log-collection/operator"
	"github.com/open-telemetry/opentelemetry-log-collection/operator/helper"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
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
	BufferConfig        buffer.Config   `json:"buffer,omitempty" yaml:"buffer,omitempty"`
	FlusherConfig       flusher.Config  `json:"flusher,omitempty" yaml:"flusher,omitempty"`
	Credentials         string          `json:"credentials,omitempty"      yaml:"credentials,omitempty"`
	CredentialsFile     string          `json:"credentials_file,omitempty" yaml:"credentials_file,omitempty"`
	ProjectID           string          `json:"project_id"                 yaml:"project_id"`
	LogNameField        *entry.Field    `json:"log_name_field,omitempty"   yaml:"log_name_field,omitempty"`
	LocationField       *entry.Field    `json:"location_field,omitempty"   yaml:"location_field,omitempty"`
	TraceField          *entry.Field    `json:"trace_field,omitempty"      yaml:"trace_field,omitempty"`
	SpanIDField         *entry.Field    `json:"span_id_field,omitempty"    yaml:"span_id_field,omitempty"`
	Timeout             helper.Duration `json:"timeout,omitempty"          yaml:"timeout,omitempty"`
	UseCompression      bool            `json:"use_compression,omitempty"  yaml:"use_compression,omitempty"`
}

// Build will build a google cloud output operator.
func (c GoogleCloudOutputConfig) Build(bc operator.BuildContext) ([]operator.Operator, error) {
	outputOperator, err := c.OutputConfig.Build(bc)
	if err != nil {
		return nil, err
	}

	newBuffer, err := c.BufferConfig.Build()
	if err != nil {
		return nil, err
	}

	newFlusher := c.FlusherConfig.Build(bc.Logger.SugaredLogger)
	ctx, cancel := context.WithCancel(context.Background())

	googleCloudOutput := &GoogleCloudOutput{
		OutputOperator:  outputOperator,
		credentials:     c.Credentials,
		credentialsFile: c.CredentialsFile,
		projectID:       c.ProjectID,
		buffer:          newBuffer,
		flusher:         newFlusher,
		logNameField:    c.LogNameField,
		locationField:   c.LocationField,
		traceField:      c.TraceField,
		spanIDField:     c.SpanIDField,
		timeout:         c.Timeout.Raw(),
		useCompression:  c.UseCompression,
		ctx:             ctx,
		cancel:          cancel,
	}

	return []operator.Operator{googleCloudOutput}, nil
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
	locationField  *entry.Field
	traceField     *entry.Field
	spanIDField    *entry.Field
	useCompression bool

	client  *vkit.Client
	timeout time.Duration

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// Start will start the google cloud logger.
func (g *GoogleCloudOutput) Start(_ operator.Persister) error {
	var credentials *google.Credentials
	var err error
	scope := "https://www.googleapis.com/auth/logging.write"
	switch {
	case g.credentials != "" && g.credentialsFile != "":
		return otelerrors.NewError("at most one of credentials or credentials_file can be configured", "")
	case g.credentials != "":
		credentials, err = google.CredentialsFromJSON(context.Background(), []byte(g.credentials), scope)
		if err != nil {
			return fmt.Errorf("parse credentials: %s", err)
		}
	case g.credentialsFile != "":
		credentialsBytes, err := ioutil.ReadFile(g.credentialsFile)
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

	if g.projectID == "" && credentials.ProjectID == "" {
		return fmt.Errorf("no project id found on google creds")
	}

	if g.projectID == "" {
		g.projectID = credentials.ProjectID
	}

	options := make([]option.ClientOption, 0, 2)
	options = append(options, option.WithCredentials(credentials))
	options = append(options, option.WithUserAgent("StanzaLogAgent/"+version.GetVersion()))
	if g.useCompression {
		options = append(options, option.WithGRPCDialOption(grpc.WithDefaultCallOptions(grpc.UseCompressor(gzip.Name))))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := vkit.NewClient(ctx, options...)
	if err != nil {
		return fmt.Errorf("create client: %w", err)
	}
	g.client = client

	// Test writing a log message
	ctx, cancel = context.WithTimeout(context.Background(), g.timeout)
	defer cancel()
	err = g.testConnection(ctx)
	if err != nil {
		return err
	}

	g.startFlushing()
	return nil
}

func (g *GoogleCloudOutput) startFlushing() {
	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		g.feedFlusher(g.ctx)
	}()
}

// Stop will flush the google cloud logger and close the underlying connection
func (g *GoogleCloudOutput) Stop() error {
	g.Debug("stopping")
	g.cancel()
	g.wg.Wait()
	g.flusher.Stop()
	// TODO handle buffer close entries
	_, err := g.buffer.Close()
	if err != nil {
		return err
	}
	if g.client != nil {
		return g.client.Close()
	}
	g.Debug("stopped")
	return nil
}

// Process processes an entry
func (g *GoogleCloudOutput) Process(ctx context.Context, e *entry.Entry) error {
	return g.buffer.Add(ctx, e)
}

// testConnection will attempt to send a test entry to google cloud logging
func (g *GoogleCloudOutput) testConnection(ctx context.Context) error {
	g.Debug("performing test connection")
	testEntry := entry.New()
	testEntry.Body = map[string]interface{}{"message": "Test connection"}
	req := g.createWriteRequest([]*entry.Entry{testEntry})
	if _, err := g.client.WriteLogEntries(ctx, req); err != nil {
		return fmt.Errorf("test connection: %s", err)
	}
	g.Debug("test connection successful")
	return nil
}

func (g *GoogleCloudOutput) feedFlusher(ctx context.Context) {
	for {
		entries, err := g.buffer.Read(ctx)
		switch {
		case errors.Is(err, context.Canceled):
			return
		case err != nil:
			g.Errorf("Failed to read chunk", zap.Error(err))
			continue
		}

		g.Debugf("processing %d entries", len(entries))
		g.flusher.Do(func(ctx context.Context) error {
			return (&splittingSender{g}).Send(ctx, entries, 0)
		})
	}
}

type bruteSender interface {
	Send(context.Context, []*entry.Entry) error
	IsTooLargeError(error) bool
	Debugf(template string, args ...interface{})
	Debugw(template string, args ...interface{})
}

func (g *GoogleCloudOutput) Send(ctx context.Context, entries []*entry.Entry) error {
	g.Debugf("sending %d entries", len(entries))
	req := g.createWriteRequest(entries)
	_, err := g.client.WriteLogEntries(ctx, req)
	if err != nil {
		return err
	}
	g.Debugf("successfully sent %d entries", len(entries))
	return nil
}

func (g *GoogleCloudOutput) IsTooLargeError(err error) bool {
	return strings.Contains(err.Error(), "exceeds maximum size") ||
		strings.Contains(err.Error(), "Maximum size exceeded")
}

type splittingSender struct {
	bruteSender
}

func (s splittingSender) Send(ctx context.Context, entries []*entry.Entry, offset uint) error {
	numEnts := len(entries)

	err := s.bruteSender.Send(ctx, entries)
	// successful send
	if err == nil {
		return nil
	}
	// We do have an error check to see if it's due to being to large a request.
	if !s.IsTooLargeError(err) {
		return err
	}

	if numEnts == 1 {
		s.Debugw("single entry too large: %s", entries[0], zap.Any("error", err))
		entries[0].Body = err.Error()
		return s.Send(ctx, entries, offset)
	}

	s.Debugf("entries too large, attempting to split them: %s", err)

	mid := numEnts / 2
	errLeft := s.Send(ctx, entries[0:mid], offset)
	errRight := s.Send(ctx, entries[mid:numEnts], offset+uint(mid))
	return multierr.Combine(errLeft, errRight)
}

func (g *GoogleCloudOutput) createWriteRequest(entries []*entry.Entry) *logpb.WriteLogEntriesRequest {
	pbEntries := make([]*logpb.LogEntry, 0, len(entries))
	for _, entry := range entries {
		pbEntry, err := g.createProtobufEntry(entry)
		if err != nil {
			g.Errorw("Failed to create protobuf entry. Dropping entry", zap.Any("error", err))
			continue
		}
		pbEntries = append(pbEntries, pbEntry)
	}

	return &logpb.WriteLogEntriesRequest{
		LogName:  g.toLogNamePath("default"),
		Entries:  pbEntries,
		Resource: g.defaultResource(),
	}
}

func (g *GoogleCloudOutput) defaultResource() *mrpb.MonitoredResource {
	return &mrpb.MonitoredResource{
		Type: "global",
		Labels: map[string]string{
			"project_id": g.projectID,
		},
	}
}

func (g *GoogleCloudOutput) toLogNamePath(logName string) string {
	return fmt.Sprintf("projects/%s/logs/%s", g.projectID, url.PathEscape(logName))
}

func (g *GoogleCloudOutput) createProtobufEntry(e *entry.Entry) (newEntry *logpb.LogEntry, err error) {
	ts, err := ptypes.TimestampProto(e.Timestamp)
	if err != nil {
		return nil, err
	}

	newEntry = &logpb.LogEntry{
		Timestamp: ts,
		Labels:    e.Attributes,
	}

	if g.logNameField != nil {
		var rawLogName string
		err := e.Read(*g.logNameField, &rawLogName)
		if err != nil {
			g.Warnw("Failed to set log name", zap.Error(err), "entry", e)
		} else {
			newEntry.LogName = g.toLogNamePath(rawLogName)
			e.Delete(*g.logNameField)
		}
	}

	if g.traceField != nil {
		err := e.Read(*g.traceField, &newEntry.Trace)
		if err != nil {
			g.Warnw("Failed to set trace", zap.Error(err), "entry", e)
		} else {
			e.Delete(*g.traceField)
		}
	}

	if g.spanIDField != nil {
		err := e.Read(*g.spanIDField, &newEntry.SpanId)
		if err != nil {
			g.Warnw("Failed to set span ID", zap.Error(err), "entry", e)
		} else {
			e.Delete(*g.spanIDField)
		}
	}

	newEntry.Severity = convertSeverity(e.Severity)
	err = setPayload(newEntry, e.Body)
	if err != nil {
		return nil, errors.Wrap(err, "set entry payload")
	}

	newEntry.Resource = getResource(e)

	if g.locationField != nil && newEntry.Resource != nil {
		var rawLocation string
		err := e.Read(*g.locationField, &rawLocation)
		if err != nil {
			g.Warnw("Failed to set location", zap.Error(err), "entry", e)
		} else {
			newEntry.Resource.Labels["location"] = rawLocation
			e.Delete(*g.locationField)
		}
	}

	// Google monitored resources wipe out Stanza's entry.Resources with
	// a static set of resources, therefore we need to move the entry's resources
	// to entry.Labels
	if newEntry.Labels == nil {
		newEntry.Labels = make(map[string]string)
	}
	for k, v := range e.Resource {
		if val, ok := newEntry.Labels[k]; ok {
			if val != v {
				g.Warnf("resource to labels merge failed, entry has label %s=%s, tried to add %s=%s", k, val, k, v)
			}
			continue
		}
		newEntry.Labels[k] = v
	}

	return newEntry, nil
}
