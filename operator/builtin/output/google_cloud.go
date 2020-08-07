package output

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"time"

	vkit "cloud.google.com/go/logging/apiv2"
	"github.com/golang/protobuf/ptypes"
	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/observiq/carbon/entry"
	"github.com/observiq/carbon/internal/version"
	"github.com/observiq/carbon/operator"
	"github.com/observiq/carbon/operator/buffer"
	"github.com/observiq/carbon/operator/helper"
	"go.uber.org/zap"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	mrpb "google.golang.org/genproto/googleapis/api/monitoredres"
	sev "google.golang.org/genproto/googleapis/logging/type"
	logpb "google.golang.org/genproto/googleapis/logging/v2"
)

func init() {
	operator.Register("google_cloud_output", func() operator.Builder { return NewGoogleCloudOutputConfig("") })
}

func NewGoogleCloudOutputConfig(operatorID string) *GoogleCloudOutputConfig {
	return &GoogleCloudOutputConfig{
		OutputConfig: helper.NewOutputConfig(operatorID, "google_cloud_output"),
		BufferConfig: buffer.NewConfig(),
		Timeout:      operator.Duration{Duration: 10 * time.Second},
	}
}

// GoogleCloudOutputConfig is the configuration of a google cloud output operator.
type GoogleCloudOutputConfig struct {
	helper.OutputConfig `yaml:",inline"`
	BufferConfig        buffer.Config `json:"buffer,omitempty" yaml:"buffer,omitempty"`

	Credentials     string            `json:"credentials,omitempty"      yaml:"credentials,omitempty"`
	CredentialsFile string            `json:"credentials_file,omitempty" yaml:"credentials_file,omitempty"`
	ProjectID       string            `json:"project_id"                 yaml:"project_id"`
	LogNameField    *entry.Field      `json:"log_name_field,omitempty"   yaml:"log_name_field,omitempty"`
	TraceField      *entry.Field      `json:"trace_field,omitempty"      yaml:"trace_field,omitempty"`
	SpanIDField     *entry.Field      `json:"span_id_field,omitempty"    yaml:"span_id_field,omitempty"`
	Timeout         operator.Duration `json:"timeout,omitempty"          yaml:"timeout,omitempty"`
}

// Build will build a google cloud output operator.
func (c GoogleCloudOutputConfig) Build(buildContext operator.BuildContext) (operator.Operator, error) {
	outputOperator, err := c.OutputConfig.Build(buildContext)
	if err != nil {
		return nil, err
	}

	newBuffer, err := c.BufferConfig.Build()
	if err != nil {
		return nil, err
	}

	googleCloudOutput := &GoogleCloudOutput{
		OutputOperator:  outputOperator,
		credentials:     c.Credentials,
		credentialsFile: c.CredentialsFile,
		projectID:       c.ProjectID,
		Buffer:          newBuffer,
		logNameField:    c.LogNameField,
		traceField:      c.TraceField,
		spanIDField:     c.SpanIDField,
		timeout:         c.Timeout.Raw(),
	}

	newBuffer.SetHandler(googleCloudOutput)

	return googleCloudOutput, nil
}

// GoogleCloudOutput is an operator that sends logs to google cloud logging.
type GoogleCloudOutput struct {
	helper.OutputOperator
	buffer.Buffer

	credentials     string
	credentialsFile string
	projectID       string

	logNameField *entry.Field
	traceField   *entry.Field
	spanIDField  *entry.Field

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

	if p.projectID == "" && credentials.ProjectID == "" {
		return fmt.Errorf("no project id found on google creds")
	}

	if p.projectID == "" {
		p.projectID = credentials.ProjectID
	}

	options := make([]option.ClientOption, 0, 2)
	options = append(options, option.WithCredentials(credentials))
	options = append(options, option.WithUserAgent("CarbonLogAgent/"+version.GetVersion()))
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
		Resource: globalResource(p.projectID),
	}

	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()
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

	newEntry.Severity = interpretSeverity(e.Severity)

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
	case map[string]string:
		fields := map[string]*structpb.Value{}
		for k, v := range p {
			fields[k] = jsonValueToStructValue(v)
		}
		newEntry.Payload = &logpb.LogEntry_JsonPayload{JsonPayload: &structpb.Struct{Fields: fields}}
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
	case int:
		return &structpb.Value{Kind: &structpb.Value_NumberValue{NumberValue: float64(x)}}
	case int64:
		return &structpb.Value{Kind: &structpb.Value_NumberValue{NumberValue: float64(x)}}
	case int32:
		return &structpb.Value{Kind: &structpb.Value_NumberValue{NumberValue: float64(x)}}
	case uint:
		return &structpb.Value{Kind: &structpb.Value_NumberValue{NumberValue: float64(x)}}
	case uint32:
		return &structpb.Value{Kind: &structpb.Value_NumberValue{NumberValue: float64(x)}}
	case uint64:
		return &structpb.Value{Kind: &structpb.Value_NumberValue{NumberValue: float64(x)}}
	case string:
		return &structpb.Value{Kind: &structpb.Value_StringValue{StringValue: x}}
	case nil:
		return &structpb.Value{Kind: &structpb.Value_NullValue{}}
	case map[string]interface{}:
		return &structpb.Value{Kind: &structpb.Value_StructValue{StructValue: jsonMapToProtoStruct(x)}}
	case map[string]map[string]string:
		fields := map[string]*structpb.Value{}
		for k, v := range x {
			fields[k] = jsonValueToStructValue(v)
		}
		return &structpb.Value{Kind: &structpb.Value_StructValue{StructValue: &structpb.Struct{Fields: fields}}}
	case map[string]string:
		fields := map[string]*structpb.Value{}
		for k, v := range x {
			fields[k] = &structpb.Value{Kind: &structpb.Value_StringValue{StringValue: v}}
		}
		return &structpb.Value{Kind: &structpb.Value_StructValue{StructValue: &structpb.Struct{Fields: fields}}}
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

var fastSev = map[entry.Severity]sev.LogSeverity{
	entry.Catastrophe: sev.LogSeverity_EMERGENCY,
	entry.Emergency:   sev.LogSeverity_EMERGENCY,
	entry.Alert:       sev.LogSeverity_ALERT,
	entry.Critical:    sev.LogSeverity_CRITICAL,
	entry.Error:       sev.LogSeverity_ERROR,
	entry.Warning:     sev.LogSeverity_WARNING,
	entry.Notice:      sev.LogSeverity_NOTICE,
	entry.Info:        sev.LogSeverity_INFO,
	entry.Debug:       sev.LogSeverity_DEBUG,
	entry.Trace:       sev.LogSeverity_DEBUG,
	entry.Default:     sev.LogSeverity_DEFAULT,
}

func interpretSeverity(s entry.Severity) sev.LogSeverity {
	if logSev, ok := fastSev[s]; ok {
		return logSev
	}

	switch {
	case s >= entry.Emergency:
		return sev.LogSeverity_EMERGENCY
	case s >= entry.Alert:
		return sev.LogSeverity_ALERT
	case s >= entry.Critical:
		return sev.LogSeverity_CRITICAL
	case s >= entry.Error:
		return sev.LogSeverity_ERROR
	case s >= entry.Warning:
		return sev.LogSeverity_WARNING
	case s >= entry.Notice:
		return sev.LogSeverity_NOTICE
	case s >= entry.Info:
		return sev.LogSeverity_INFO
	case s > entry.Default:
		return sev.LogSeverity_DEBUG
	default:
		return sev.LogSeverity_DEFAULT
	}
}
