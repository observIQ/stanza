package builtin

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"strings"
	"time"

	vkit "cloud.google.com/go/logging/apiv2"
	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/buffer"
	"github.com/bluemedora/bplogagent/plugin/helper"
	"github.com/golang/protobuf/ptypes"
	structpb "github.com/golang/protobuf/ptypes/struct"
	gax "github.com/googleapis/gax-go"
	"go.uber.org/zap"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	mrpb "google.golang.org/genproto/googleapis/api/monitoredres"
	sev "google.golang.org/genproto/googleapis/logging/type"
	logpb "google.golang.org/genproto/googleapis/logging/v2"
)

func init() {
	plugin.Register("google_cloud_output", &GoogleCloudOutputConfig{})
}

// GoogleCloudOutputConfig is the configuration of a google cloud output plugin.
type GoogleCloudOutputConfig struct {
	helper.OutputConfig `yaml:",inline"`
	buffer.BufferConfig `json:"buffer,omitempty" yaml:"buffer,omitempty"`

	Credentials     string       `json:"credentials,omitempty"      yaml:"credentials,omitempty"`
	CredentialsFile string       `json:"credentials_file,omitempty" yaml:"credentials_file,omitempty"`
	ProjectID       string       `json:"project_id"                 yaml:"project_id"`
	LogNameField    *entry.Field `json:"log_name_field,omitempty"   yaml:"log_name_field,omitempty"`
	LabelsField     *entry.Field `json:"labels_field,omitempty"     yaml:"labels_field,omitempty"`
	SeverityField   *entry.Field `json:"severity_field,omitempty"   yaml:"severity_field,omitempty"`
	TraceField      *entry.Field `json:"trace_field,omitempty"      yaml:"trace_field,omitempty"`
	SpanIDField     *entry.Field `json:"span_id_field,omitempty"    yaml:"span_id_field,omitempty"`
}

// Build will build a google cloud output plugin.
func (c GoogleCloudOutputConfig) Build(buildContext plugin.BuildContext) (plugin.Plugin, error) {
	outputPlugin, err := c.OutputConfig.Build(buildContext)
	if err != nil {
		return nil, err
	}

	if c.ProjectID == "" {
		return nil, errors.New("missing required configuration option project_id")
	}

	newBuffer, err := c.BufferConfig.Build()
	if err != nil {
		return nil, err
	}

	googleCloudOutput := &GoogleCloudOutput{
		OutputPlugin:    outputPlugin,
		credentials:     c.Credentials,
		credentialsFile: c.CredentialsFile,
		projectID:       c.ProjectID,
		Buffer:          newBuffer,
		logNameField:    c.LogNameField,
		labelsField:     c.LabelsField,
		severityField:   c.SeverityField,
		traceField:      c.TraceField,
		spanIDField:     c.SpanIDField,
	}

	newBuffer.SetHandler(googleCloudOutput)

	return googleCloudOutput, nil
}

// GoogleCloudOutput is a plugin that sends logs to google cloud logging.
type GoogleCloudOutput struct {
	helper.OutputPlugin
	buffer.Buffer

	credentials     string
	credentialsFile string
	projectID       string

	logNameField  *entry.Field
	labelsField   *entry.Field
	severityField *entry.Field
	traceField    *entry.Field
	spanIDField   *entry.Field

	client CloudLoggingClient
}

type CloudLoggingClient interface {
	Close() error
	WriteLogEntries(context.Context, *logpb.WriteLogEntriesRequest, ...gax.CallOption) (*logpb.WriteLogEntriesResponse, error)
}

// Start will start the google cloud logger.
func (p *GoogleCloudOutput) Start() error {
	var credentials *google.Credentials
	var err error
	scope := "https://www.googleapis.com/auth/logging.write"
	switch {
	case p.credentials != "" && p.credentialsFile != "":
		return errors.New("at most one of credentials or credentials_file can be configured")
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

	options := make([]option.ClientOption, 0, 2)
	options = append(options, option.WithCredentials(credentials))
	options = append(options, option.WithUserAgent("BindPlaneLogAgent/2.0.0"))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := vkit.NewClient(ctx, options...)
	if err != nil {
		return fmt.Errorf("create client: %w", err)
	}
	p.client = client

	// Test writing a log message
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	testEntry := entry.New()
	testEntry.Record = map[string]interface{}{"message": "Test connection"}
	err = p.ProcessMulti(ctx, []*entry.Entry{testEntry})
	if err != nil {
		return fmt.Errorf("test connection: %s", err)
	}

	return nil
}

// Stop will flush the google cloud logger and close the underlying connection
func (p *GoogleCloudOutput) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err := p.Buffer.Flush(ctx)
	if err != nil {
		p.Warnw("Failed to flush", zap.Error(err))
	}
	return p.client.Close()
}

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
		Resource: globalResource(p.projectID),
	}

	_, err := p.client.WriteLogEntries(ctx, &req)
	if err != nil {
		return fmt.Errorf("write log entries: %s", err)
	}

	return nil
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

	if p.labelsField != nil {
		err := e.Read(*p.labelsField, &newEntry.Labels)
		if err != nil {
			p.Warnw("Failed to set labels", zap.Error(err), "entry", e)
		} else {
			e.Delete(*p.labelsField)
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

	if p.severityField != nil {
		var severityString string
		err := e.Read(*p.severityField, &severityString)
		if err != nil {
			p.Warnw("Failed to set severity", zap.Error(err), "entry", e)
		} else {
			e.Delete(*p.severityField)
		}
		newEntry.Severity, err = parseSeverity(severityString)
		if err != nil {
			p.Warnw("Failed to parse severity", zap.Error(err), "entry", e)
		}
	}

	// Protect against the panic condition inside `jsonValueToStructValue`
	defer func() {
		if r := recover(); r != nil {
			newEntry = nil
			err = fmt.Errorf(r.(string))
		}
	}()
	switch p := e.Record.(type) {
	case string:
		newEntry.Payload = &logpb.LogEntry_TextPayload{TextPayload: p}
	case []byte:
		newEntry.Payload = &logpb.LogEntry_TextPayload{TextPayload: string(p)}
	case map[string]interface{}:
		s := jsonMapToProtoStruct(p)
		newEntry.Payload = &logpb.LogEntry_JsonPayload{JsonPayload: s}
	default:
		return nil, fmt.Errorf("cannot convert record of type %T to a protobuf representation", e.Record)
	}

	return newEntry, nil
}

func jsonMapToProtoStruct(m map[string]interface{}) *structpb.Struct {
	fields := map[string]*structpb.Value{}
	for k, v := range m {
		fields[k] = jsonValueToStructValue(v)
	}
	return &structpb.Struct{Fields: fields}
}

func jsonValueToStructValue(v interface{}) *structpb.Value {
	switch x := v.(type) {
	case bool:
		return &structpb.Value{Kind: &structpb.Value_BoolValue{BoolValue: x}}
	case float64:
		return &structpb.Value{Kind: &structpb.Value_NumberValue{NumberValue: x}}
	case string:
		return &structpb.Value{Kind: &structpb.Value_StringValue{StringValue: x}}
	case nil:
		return &structpb.Value{Kind: &structpb.Value_NullValue{}}
	case map[string]interface{}:
		return &structpb.Value{Kind: &structpb.Value_StructValue{StructValue: jsonMapToProtoStruct(x)}}
	case []interface{}:
		var vals []*structpb.Value
		for _, e := range x {
			vals = append(vals, jsonValueToStructValue(e))
		}
		return &structpb.Value{Kind: &structpb.Value_ListValue{ListValue: &structpb.ListValue{Values: vals}}}
	default:
		panic(fmt.Sprintf("bad type %T for JSON value", v))
	}
}

func globalResource(projectID string) *mrpb.MonitoredResource {
	return &mrpb.MonitoredResource{
		Type: "global",
		Labels: map[string]string{
			"project_id": projectID,
		},
	}
}

func parseSeverity(severity string) (sev.LogSeverity, error) {
	val, ok := sev.LogSeverity_value[strings.ToUpper(severity)]
	if !ok {
		return sev.LogSeverity_DEFAULT, fmt.Errorf("unknown severity '%s'", severity)
	}

	return sev.LogSeverity(val), nil
}
