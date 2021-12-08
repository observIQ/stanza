package googlecloud

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/observiq/stanza/entry"

	pstruct "github.com/golang/protobuf/ptypes/struct"
	"google.golang.org/genproto/googleapis/logging/v2"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// EntryBuilder is an interface for building google cloud logging entries
type EntryBuilder interface {
	Build(entry *entry.Entry) (*logging.LogEntry, error)
}

// GoogleEntryBuilder is used to build google cloud logging entries
type GoogleEntryBuilder struct {
	MaxEntrySize  int
	ProjectID     string
	LogNameField  *entry.Field
	LocationField *entry.Field
	TraceField    *entry.Field
	SpanIDField   *entry.Field
}

// Build builds a google cloud logging entry from a stanza entry.
// The size of the resulting entry is also returned.
func (g *GoogleEntryBuilder) Build(entry *entry.Entry) (*logging.LogEntry, error) {
	logEntry := &logging.LogEntry{
		Timestamp: timestamppb.New(entry.Timestamp),
		Resource:  createResource(entry),
		Severity:  convertSeverity(entry.Severity),
	}

	if err := g.setLogName(entry, logEntry); err != nil {
		return nil, fmt.Errorf("failed to set log name: %w", err)
	}

	if err := g.setTrace(entry, logEntry); err != nil {
		return nil, fmt.Errorf("failed to set trace: %w", err)
	}

	if err := g.setSpanID(entry, logEntry); err != nil {
		return nil, fmt.Errorf("failed to set span id: %w", err)
	}

	if err := g.setLocation(entry, logEntry); err != nil {
		return nil, fmt.Errorf("failed to set location: %w", err)
	}

	if err := g.setLabels(entry, logEntry); err != nil {
		return nil, fmt.Errorf("failed to set labels: %w", err)
	}

	if err := g.setPayload(entry, logEntry); err != nil {
		return nil, fmt.Errorf("failed to set payload: %w", err)
	}

	protoSize := proto.Size(logEntry)
	if protoSize > g.MaxEntrySize {
		return nil, fmt.Errorf("exceeds max entry size: %d", protoSize)
	}

	return logEntry, nil
}

// setLogName sets the log name of the google log entry using a field on the stanza entry
func (g *GoogleEntryBuilder) setLogName(entry *entry.Entry, logEntry *logging.LogEntry) error {
	if g.LogNameField == nil {
		return nil
	}

	var value string
	err := entry.Read(*g.LogNameField, &value)
	if err != nil {
		return fmt.Errorf("failed to read log name field: %w", err)
	}

	logEntry.LogName = createLogName(g.ProjectID, value)
	entry.Delete(*g.LogNameField)

	return nil
}

// setTrace sets the trace of the protobuf entry using a field on the stanza entry
func (g *GoogleEntryBuilder) setTrace(entry *entry.Entry, logEntry *logging.LogEntry) error {
	if g.TraceField == nil {
		return nil
	}

	err := entry.Read(*g.TraceField, &logEntry.Trace)
	if err != nil {
		return fmt.Errorf("failed to read trace field: %w", err)
	}

	entry.Delete(*g.TraceField)
	return nil
}

// setSpanID sets the span id of the protobuf entry using a field on the stanza entry
func (g *GoogleEntryBuilder) setSpanID(entry *entry.Entry, logEntry *logging.LogEntry) error {
	if g.SpanIDField == nil {
		return nil
	}

	err := entry.Read(*g.SpanIDField, &logEntry.SpanId)
	if err != nil {
		return fmt.Errorf("failed to read span id field: %w", err)
	}

	entry.Delete(*g.SpanIDField)
	return nil
}

// setLocation sets the location of the protobuf entry using a field on the stanza entry
func (g *GoogleEntryBuilder) setLocation(entry *entry.Entry, logEntry *logging.LogEntry) error {
	if g.LocationField == nil {
		return nil
	}

	if logEntry.Resource == nil {
		return errors.New("resource is nil")
	}

	var value string
	err := entry.Read(*g.LocationField, &value)
	if err != nil {
		return fmt.Errorf("failed to read location field: %w", err)
	}

	logEntry.Resource.Labels["location"] = value
	entry.Delete(*g.LocationField)

	return nil
}

// setLabels sets the labels of the protobuf entry based on the supplied stanza entry
func (g *GoogleEntryBuilder) setLabels(entry *entry.Entry, logEntry *logging.LogEntry) error {
	labels := entry.Labels
	if labels == nil {
		labels = make(map[string]string)
	}

	for key, value := range entry.Resource {
		if _, ok := labels[key]; ok {
			return fmt.Errorf("duplicate key exists on both labels and resource: %s", key)
		}
		labels[key] = value
	}

	logEntry.Labels = labels
	return nil
}

// setPayload sets the payload of the protobuf entry based on the supplied stanza entry
func (g *GoogleEntryBuilder) setPayload(entry *entry.Entry, logEntry *logging.LogEntry) error {
	switch value := entry.Record.(type) {
	case string:
		logEntry.Payload = &logging.LogEntry_TextPayload{TextPayload: value}
		return nil
	case []byte:
		logEntry.Payload = &logging.LogEntry_TextPayload{TextPayload: string(value)}
		return nil
	case map[string]interface{}:
		structValue, err := structpb.NewValue(value)
		if err != nil {
			return fmt.Errorf("failed to convert record of type map[string]interface: %w", err)
		}

		logEntry.Payload = &logging.LogEntry_JsonPayload{JsonPayload: structValue.GetStructValue()}
		return nil
	case map[string]string:
		fields := map[string]*pstruct.Value{}
		for k, v := range value {
			fields[k] = &pstruct.Value{Kind: &pstruct.Value_StringValue{StringValue: v}}
		}

		logEntry.Payload = &logging.LogEntry_JsonPayload{JsonPayload: &pstruct.Struct{Fields: fields}}
		return nil
	default:
		return fmt.Errorf("cannot convert record of type %T", entry.Record)
	}
}

// createLogName creates a log name from the supplied project id and name value
func createLogName(projectID, name string) string {
	return fmt.Sprintf("projects/%s/logs/%s", projectID, url.PathEscape(name))
}
