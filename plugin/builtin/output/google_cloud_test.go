package output

import (
	"context"
	"testing"
	"time"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/buffer"
	"github.com/bluemedora/bplogagent/plugin/helper"
	"github.com/bluemedora/bplogagent/plugin/testutil"
	"github.com/golang/protobuf/ptypes"
	gax "github.com/googleapis/gax-go"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/api/monitoredres"
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

func TestGoogleCloudOutput(t *testing.T) {
	basicConfig := func() *GoogleCloudOutputConfig {
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

	basicWriteEntriesRequest := func() *logpb.WriteLogEntriesRequest {
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

	now, _ := time.Parse(time.RFC3339, time.RFC3339)
	protoTs, _ := ptypes.TimestampProto(now)

	cases := []struct {
		name           string
		config         *GoogleCloudOutputConfig
		input          *entry.Entry
		expectedOutput *logpb.WriteLogEntriesRequest
	}{
		{
			"Basic",
			basicConfig(),
			&entry.Entry{
				Timestamp: now,
				Record: map[string]interface{}{
					"message": "test message",
				},
			},
			func() *logpb.WriteLogEntriesRequest {
				req := basicWriteEntriesRequest()
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
				c := basicConfig()
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
				req := basicWriteEntriesRequest()
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
			"LabelsField",
			func() *GoogleCloudOutputConfig {
				c := basicConfig()
				f := entry.NewField("labels")
				c.LabelsField = &f
				return c
			}(),
			&entry.Entry{
				Timestamp: now,
				Record: map[string]interface{}{
					"message": "test message",
					"labels": map[string]interface{}{
						"label1": "value1",
					},
				},
			},
			func() *logpb.WriteLogEntriesRequest {
				req := basicWriteEntriesRequest()
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
		{
			"LabelsField",
			func() *GoogleCloudOutputConfig {
				c := basicConfig()
				f := entry.NewField("labels")
				c.LabelsField = &f
				return c
			}(),
			&entry.Entry{
				Timestamp: now,
				Record: map[string]interface{}{
					"message": "test message",
					"labels": map[string]interface{}{
						"label1": "value1",
					},
				},
			},
			func() *logpb.WriteLogEntriesRequest {
				req := basicWriteEntriesRequest()
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
		{
			"SeverityField",
			func() *GoogleCloudOutputConfig {
				c := basicConfig()
				f := entry.NewField("severity")
				c.SeverityField = &f
				return c
			}(),
			&entry.Entry{
				Timestamp: now,
				Record: map[string]interface{}{
					"message":  "test message",
					"severity": "error",
				},
			},
			func() *logpb.WriteLogEntriesRequest {
				req := basicWriteEntriesRequest()
				req.Entries = []*logpb.LogEntry{
					{
						Severity:  500,
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
			"TraceAndSpanFields",
			func() *GoogleCloudOutputConfig {
				c := basicConfig()
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
				req := basicWriteEntriesRequest()
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
			buildContext := testutil.NewTestBuildContext(t)
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
