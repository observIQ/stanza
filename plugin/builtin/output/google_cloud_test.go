package output

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/internal/testutil"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/buffer"
	"github.com/bluemedora/bplogagent/plugin/helper"
	"github.com/golang/protobuf/ptypes"
	tspb "github.com/golang/protobuf/ptypes/timestamp"
	gax "github.com/googleapis/gax-go"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/api/monitoredres"
	sev "google.golang.org/genproto/googleapis/logging/type"
	logpb "google.golang.org/genproto/googleapis/logging/v2"
)

type mockCloudLoggingClient struct {
	closed          chan struct{}
	writeLogEntries chan *logpb.WriteLogEntriesRequest
}

func (client *mockCloudLoggingClient) Close() error {
	client.closed <- struct{}{}
	return nil
}

func (client *mockCloudLoggingClient) WriteLogEntries(ctx context.Context, req *logpb.WriteLogEntriesRequest, opts ...gax.CallOption) (*logpb.WriteLogEntriesResponse, error) {
	client.writeLogEntries <- req
	return nil, nil
}

type googleCloudTestCase struct {
	name           string
	config         *GoogleCloudOutputConfig
	input          *entry.Entry
	expectedOutput *logpb.WriteLogEntriesRequest
}

func googleCloudBasicConfig() *GoogleCloudOutputConfig {
	return &GoogleCloudOutputConfig{
		OutputConfig: helper.OutputConfig{
			BasicConfig: helper.BasicConfig{
				PluginID:   "test_id",
				PluginType: "google_cloud_output",
			},
		},
		BufferConfig: buffer.BufferConfig{
			DelayThreshold: plugin.Duration{
				Duration: time.Millisecond,
			},
		},
		ProjectID: "test_project_id",
	}
}

func googleCloudBasicWriteEntriesRequest() *logpb.WriteLogEntriesRequest {
	return &logpb.WriteLogEntriesRequest{
		LogName: "projects/test_project_id/logs/default",
		Resource: &monitoredres.MonitoredResource{
			Type: "global",
			Labels: map[string]string{
				"project_id": "test_project_id",
			},
		},
	}
}

func googleCloudTimes() (time.Time, *tspb.Timestamp) {
	now, _ := time.Parse(time.RFC3339, time.RFC3339)
	protoTs, _ := ptypes.TimestampProto(now)
	return now, protoTs
}

func TestGoogleCloudOutput(t *testing.T) {

	now, protoTs := googleCloudTimes()

	cases := []googleCloudTestCase{
		{
			"Basic",
			googleCloudBasicConfig(),
			&entry.Entry{
				Timestamp: now,
				Record: map[string]interface{}{
					"message": "test message",
				},
			},
			func() *logpb.WriteLogEntriesRequest {
				req := googleCloudBasicWriteEntriesRequest()
				req.Entries = []*logpb.LogEntry{
					{
						Timestamp: protoTs,
						Payload: &logpb.LogEntry_JsonPayload{JsonPayload: jsonMapToProtoStruct(map[string]interface{}{
							"message": "test message",
						})},
					},
				}
				return req
			}(),
		},
		{
			"LogNameField",
			func() *GoogleCloudOutputConfig {
				c := googleCloudBasicConfig()
				f := entry.NewField("log_name")
				c.LogNameField = &f
				return c
			}(),
			&entry.Entry{
				Timestamp: now,
				Record: map[string]interface{}{
					"message":  "test message",
					"log_name": "mylogname",
				},
			},
			func() *logpb.WriteLogEntriesRequest {
				req := googleCloudBasicWriteEntriesRequest()
				req.Entries = []*logpb.LogEntry{
					{
						LogName:   "projects/test_project_id/logs/mylogname",
						Timestamp: protoTs,
						Payload: &logpb.LogEntry_JsonPayload{JsonPayload: jsonMapToProtoStruct(map[string]interface{}{
							"message": "test message",
						})},
					},
				}
				return req
			}(),
		},
		{
			"Labels",
			func() *GoogleCloudOutputConfig {
				return googleCloudBasicConfig()
			}(),
			&entry.Entry{
				Timestamp: now,
				Labels: map[string]string{
					"label1": "value1",
				},
				Record: map[string]interface{}{
					"message": "test message",
				},
			},
			func() *logpb.WriteLogEntriesRequest {
				req := googleCloudBasicWriteEntriesRequest()
				req.Entries = []*logpb.LogEntry{
					{
						Labels: map[string]string{
							"label1": "value1",
						},
						Timestamp: protoTs,
						Payload: &logpb.LogEntry_JsonPayload{JsonPayload: jsonMapToProtoStruct(map[string]interface{}{
							"message": "test message",
						})},
					},
				}
				return req
			}(),
		},
		googleCloudSeverityTestCase(entry.Catastrophe, sev.LogSeverity_EMERGENCY),
		googleCloudSeverityTestCase(entry.Severity(95), sev.LogSeverity_EMERGENCY),
		googleCloudSeverityTestCase(entry.Emergency, sev.LogSeverity_EMERGENCY),
		googleCloudSeverityTestCase(entry.Severity(85), sev.LogSeverity_ALERT),
		googleCloudSeverityTestCase(entry.Alert, sev.LogSeverity_ALERT),
		googleCloudSeverityTestCase(entry.Severity(75), sev.LogSeverity_CRITICAL),
		googleCloudSeverityTestCase(entry.Critical, sev.LogSeverity_CRITICAL),
		googleCloudSeverityTestCase(entry.Severity(65), sev.LogSeverity_ERROR),
		googleCloudSeverityTestCase(entry.Error, sev.LogSeverity_ERROR),
		googleCloudSeverityTestCase(entry.Severity(55), sev.LogSeverity_WARNING),
		googleCloudSeverityTestCase(entry.Warning, sev.LogSeverity_WARNING),
		googleCloudSeverityTestCase(entry.Severity(45), sev.LogSeverity_NOTICE),
		googleCloudSeverityTestCase(entry.Notice, sev.LogSeverity_NOTICE),
		googleCloudSeverityTestCase(entry.Severity(35), sev.LogSeverity_INFO),
		googleCloudSeverityTestCase(entry.Info, sev.LogSeverity_INFO),
		googleCloudSeverityTestCase(entry.Severity(25), sev.LogSeverity_DEBUG),
		googleCloudSeverityTestCase(entry.Debug, sev.LogSeverity_DEBUG),
		googleCloudSeverityTestCase(entry.Severity(15), sev.LogSeverity_DEFAULT),
		googleCloudSeverityTestCase(entry.Trace, sev.LogSeverity_DEFAULT),
		googleCloudSeverityTestCase(entry.Severity(5), sev.LogSeverity_DEFAULT),
		googleCloudSeverityTestCase(entry.Default, sev.LogSeverity_DEFAULT),
		{
			"TraceAndSpanFields",
			func() *GoogleCloudOutputConfig {
				c := googleCloudBasicConfig()
				traceField := entry.NewField("trace")
				spanIDField := entry.NewField("span_id")
				c.TraceField = &traceField
				c.SpanIDField = &spanIDField
				return c
			}(),
			&entry.Entry{
				Timestamp: now,
				Record: map[string]interface{}{
					"message": "test message",
					"trace":   "projects/my-projectid/traces/06796866738c859f2f19b7cfb3214824",
					"span_id": "000000000000004a",
				},
			},
			func() *logpb.WriteLogEntriesRequest {
				req := googleCloudBasicWriteEntriesRequest()
				req.Entries = []*logpb.LogEntry{
					{
						Trace:     "projects/my-projectid/traces/06796866738c859f2f19b7cfb3214824",
						SpanId:    "000000000000004a",
						Timestamp: protoTs,
						Payload: &logpb.LogEntry_JsonPayload{JsonPayload: jsonMapToProtoStruct(map[string]interface{}{
							"message": "test message",
						})},
					},
				}
				return req
			}(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			buildContext := testutil.NewBuildContext(t)
			cloudOutput, err := tc.config.Build(buildContext)
			require.NoError(t, err)

			mockClient := &mockCloudLoggingClient{
				closed:          make(chan struct{}, 1),
				writeLogEntries: make(chan *logpb.WriteLogEntriesRequest, 10),
			}
			cloudOutput.(*GoogleCloudOutput).client = mockClient

			err = cloudOutput.Process(context.Background(), tc.input)
			require.NoError(t, err)

			select {
			case req := <-mockClient.writeLogEntries:
				require.Equal(t, tc.expectedOutput, req)
			case <-time.After(time.Second):
				require.FailNow(t, "Timed out waiting for writeLogEntries request")
			}
		})
	}
}

func googleCloudSeverityTestCase(s entry.Severity, expected sev.LogSeverity) googleCloudTestCase {
	now, protoTs := googleCloudTimes()
	return googleCloudTestCase{
		fmt.Sprintf("Severity%s", s),
		func() *GoogleCloudOutputConfig {
			return googleCloudBasicConfig()
		}(),
		&entry.Entry{
			Timestamp: now,
			Severity:  s,
			Record: map[string]interface{}{
				"message": "test message",
			},
		},
		func() *logpb.WriteLogEntriesRequest {
			req := googleCloudBasicWriteEntriesRequest()
			req.Entries = []*logpb.LogEntry{
				{
					Severity:  expected,
					Timestamp: protoTs,
					Payload: &logpb.LogEntry_JsonPayload{JsonPayload: jsonMapToProtoStruct(map[string]interface{}{
						"message": "test message",
					})},
				},
			}
			return req
		}(),
	}
}
