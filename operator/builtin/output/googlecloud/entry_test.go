package googlecloud

import (
	"testing"
	"time"

	"github.com/observiq/stanza/entry"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/api/monitoredres"
	"google.golang.org/genproto/googleapis/logging/v2"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestBuildEntry(t *testing.T) {
	testCases := []struct {
		name          string
		stanzaEntry   *entry.Entry
		builder       GoogleEntryBuilder
		expectedEntry *logging.LogEntry
		expectedErr   string
	}{
		{
			name:        "missing log name field",
			stanzaEntry: &entry.Entry{},
			builder: GoogleEntryBuilder{
				LogNameField: newRecordField("log_name"),
			},
			expectedErr: "failed to set log name",
		},
		{
			name:        "missing trace field",
			stanzaEntry: &entry.Entry{},
			builder: GoogleEntryBuilder{
				TraceField: newRecordField("trace"),
			},
			expectedErr: "failed to set trace",
		},
		{
			name:        "missing span id field",
			stanzaEntry: &entry.Entry{},
			builder: GoogleEntryBuilder{
				SpanIDField: newRecordField("span_id"),
			},
			expectedErr: "failed to set span id",
		},
		{
			name:        "nil resource for location",
			stanzaEntry: &entry.Entry{},
			builder: GoogleEntryBuilder{
				LocationField: newRecordField("location"),
			},
			expectedErr: "resource is nil",
		},
		{
			name: "missing location field",
			stanzaEntry: &entry.Entry{
				Resource: map[string]string{
					"host.name": "test_host",
				},
			},
			builder: GoogleEntryBuilder{
				LocationField: newRecordField("location"),
			},
			expectedErr: "failed to read location field",
		},
		{
			name: "duplicate label keys",
			stanzaEntry: &entry.Entry{
				Record: "test record",
				Labels: map[string]string{
					"duplicate_key": "value_1",
				},
				Resource: map[string]string{
					"duplicate_key": "value_2",
				},
			},
			builder:     GoogleEntryBuilder{},
			expectedErr: "failed to set labels",
		},
		{
			name:        "invalid payload",
			stanzaEntry: &entry.Entry{},
			builder:     GoogleEntryBuilder{},
			expectedErr: "failed to set payload",
		},
		{
			name: "exceeds maximum size",
			stanzaEntry: &entry.Entry{
				Record: "test record",
			},
			builder: GoogleEntryBuilder{
				MaxEntrySize: 5,
			},
			expectedErr: "exceeds max entry size",
		},
		{
			name: "invalid map entry",
			stanzaEntry: &entry.Entry{
				Record: map[string]interface{}{
					"invalid": make(chan int),
				},
			},
			builder: GoogleEntryBuilder{
				MaxEntrySize: 5000,
			},
			expectedErr: "failed to convert record of type map[string]interface",
		},
		{
			name: "valid map entry",
			stanzaEntry: &entry.Entry{
				Record: map[string]interface{}{
					"int_value":    5,
					"string_value": "test",
					"bool_value":   true,
					"log_name":     "test_log",
					"location":     "test_location",
					"trace":        "test_trace",
					"span_id":      "test_span",
				},
				Resource: map[string]string{
					"host.name":      "test_host",
					"resource_label": "resource_value",
				},
				Labels: map[string]string{
					"test_label": "test_value",
				},
				Timestamp: time.UnixMilli(0),
			},
			builder: GoogleEntryBuilder{
				MaxEntrySize:  5000,
				ProjectID:     "test_project",
				LogNameField:  newRecordField("log_name"),
				LocationField: newRecordField("location"),
				TraceField:    newRecordField("trace"),
				SpanIDField:   newRecordField("span_id"),
			},
			expectedEntry: &logging.LogEntry{
				LogName: "projects/test_project/logs/test_log",
				Payload: &logging.LogEntry_JsonPayload{
					JsonPayload: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"int_value":    structpb.NewNumberValue(5),
							"string_value": structpb.NewStringValue("test"),
							"bool_value":   structpb.NewBoolValue(true),
						},
					},
				},
				Labels: map[string]string{
					"test_label":     "test_value",
					"resource_label": "resource_value",
				},
				Trace:  "test_trace",
				SpanId: "test_span",
				Resource: &monitoredres.MonitoredResource{
					Type: genericNode,
					Labels: map[string]string{
						"node_id":  "test_host",
						"location": "test_location",
					},
				},
				Timestamp: timestamppb.New(time.UnixMilli(0)),
			},
		},
		{
			name: "valid bytes entry",
			stanzaEntry: &entry.Entry{
				Record:    []byte("test"),
				Timestamp: time.UnixMilli(0),
			},
			builder: GoogleEntryBuilder{
				MaxEntrySize: 5000,
				ProjectID:    "test_project",
			},
			expectedEntry: &logging.LogEntry{
				Payload: &logging.LogEntry_TextPayload{
					TextPayload: "test",
				},
				Timestamp: timestamppb.New(time.UnixMilli(0)),
			},
		},
		{
			name: "valid string map entry",
			stanzaEntry: &entry.Entry{
				Record: map[string]string{
					"test": "value",
				},
				Timestamp: time.UnixMilli(0),
			},
			builder: GoogleEntryBuilder{
				MaxEntrySize: 5000,
				ProjectID:    "test_project",
			},
			expectedEntry: &logging.LogEntry{
				Payload: &logging.LogEntry_JsonPayload{
					JsonPayload: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"test": structpb.NewStringValue("value"),
						},
					},
				},
				Timestamp: timestamppb.New(time.UnixMilli(0)),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			entry, err := tc.builder.Build(tc.stanzaEntry)
			if tc.expectedErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectedErr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.expectedEntry.String(), entry.String())
		})
	}
}

func newRecordField(value string) *entry.Field {
	return &entry.Field{
		FieldInterface: entry.NewRecordField(value),
	}
}
