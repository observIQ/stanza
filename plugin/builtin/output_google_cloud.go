package builtin

import (
	"context"
	"errors"
	"fmt"
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
	"go.uber.org/zap"
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
	helper.BasicPluginConfig `mapstructure:",squash" yaml:",inline"`

	Credentials   string      `mapstructure:"credentials"    json:"credentials"              yaml:"credentials"`
	ProjectID     string      `mapstructure:"project_id"     json:"project_id"               yaml:"project_id"`
	LogNameField  entry.Field `mapstructure:"log_name_field" json:"log_name_field,omitempty" yaml:"log_name_field,omitempty,flow"`
	LabelsField   entry.Field `mapstructure:"labels_field"   json:"labels_field,omitempty"   yaml:"labels_field,omitempty,flow"`
	SeverityField entry.Field `mapstructure:"severity_field" json:"severity_field,omitempty" yaml:"severity_field,omitempty,flow"`
	TraceField    entry.Field `mapstructure:"trace_field"    json:"trace_field,omitempty"    yaml:"trace_field,omitempty,flow"`
	SpanIDField   entry.Field `mapstructure:"span_id_field"  json:"span_id_field,omitempty"  yaml:"span_id_field,omitempty,flow"`
}

// Build will build a google cloud output plugin.
func (c GoogleCloudOutputConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	basicPlugin, err := c.BasicPluginConfig.Build(context.Logger)
	if err != nil {
		return nil, err
	}

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

// GoogleCloudOutput is a plugin that sends logs to google cloud logging.
type GoogleCloudOutput struct {
	helper.BasicPlugin
	helper.BasicOutput

	credentials string
	projectID   string

	logNameField  entry.Field
	labelsField   entry.Field
	severityField entry.Field
	traceField    entry.Field
	spanIDField   entry.Field

	buffer buffer.Buffer
	client *vkit.Client
}

// Start will start the google cloud logger.
func (p *GoogleCloudOutput) Start() error {
	options := make([]option.ClientOption, 0, 2)
	options = append(options, option.WithCredentialsJSON([]byte(p.credentials)))
	options = append(options, option.WithUserAgent("BindPlaneLogAgent/2.0.0"))
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*10))
	defer cancel()

	client, err := vkit.NewClient(ctx, options...)
	if err != nil {
		return fmt.Errorf("create client: %w", err)
	}
	p.client = client

	p.buffer = buffer.NewMemoryBuffer(&logpb.LogEntry{}, func(ctx context.Context, entries interface{}) error {
		castEntries := entries.([]*logpb.LogEntry)
		err := p.writeEntries(ctx, castEntries)
		if err != nil {
			p.Warnw("Failed to flush", zap.Error(err))
			return err
		}
		return nil
	})

	return nil
}

// Stop will flush the google cloud logger and close the underlying connection
func (p *GoogleCloudOutput) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err := p.buffer.Flush(ctx)
	if err != nil {
		p.Warnw("Failed to flush", zap.Error(err))
	}
	return p.client.Close()
}

// Process will send an entry to google cloud logging.
func (p *GoogleCloudOutput) Process(entry *entry.Entry) error {
	pbEntry, err := p.createProtobufEntry(entry)
	if err != nil {
		return err
	}

	return p.buffer.AddWait(context.TODO(), pbEntry, 0)
}

func (p *GoogleCloudOutput) writeEntries(ctx context.Context, entries []*logpb.LogEntry) error {
	req := logpb.WriteLogEntriesRequest{
		LogName:  p.toLogNamePath("default"),
		Entries:  entries,
		Resource: globalResource(p.projectID),
	}

	_, err := p.client.WriteLogEntries(ctx, &req)
	if err != nil {
		return err
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
		err := e.Read(p.logNameField, &rawLogName)
		if err != nil {
			p.Warnw("Failed to set log name", zap.Error(err), "entry", e)
		} else {
			newEntry.LogName = p.toLogNamePath(rawLogName)
			e.Delete(p.logNameField)
		}
	}

	if p.labelsField != nil {
		err := e.Read(p.labelsField, &newEntry.Labels)
		if err != nil {
			p.Warnw("Failed to set labels", zap.Error(err), "entry", e)
		} else {
			e.Delete(p.labelsField)
		}
	}

	if p.traceField != nil {
		err := e.Read(p.traceField, &newEntry.Trace)
		if err != nil {
			p.Warnw("Failed to set trace", zap.Error(err), "entry", e)
		} else {
			e.Delete(p.traceField)
		}
	}

	if p.spanIDField != nil {
		err := e.Read(p.spanIDField, &newEntry.SpanId)
		if err != nil {
			p.Warnw("Failed to set span ID", zap.Error(err), "entry", e)
		} else {
			e.Delete(p.spanIDField)
		}
	}

	if p.severityField != nil {
		var severityString string
		err := e.Read(p.severityField, &severityString)
		if err != nil {
			p.Warnw("Failed to set severity", zap.Error(err), "entry", e)
		} else {
			e.Delete(p.severityField)
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
