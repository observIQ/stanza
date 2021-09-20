package googlecloud

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"sync"
	"testing"
	"time"

	vkit "cloud.google.com/go/logging/apiv2"
	"github.com/golang/protobuf/ptypes"
	tspb "github.com/golang/protobuf/ptypes/timestamp"
	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator/buffer"
	"github.com/observiq/stanza/operator/helper"
	"github.com/observiq/stanza/testutil"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/option"
	"google.golang.org/genproto/googleapis/api/monitoredres"
	sev "google.golang.org/genproto/googleapis/logging/type"
	logpb "google.golang.org/genproto/googleapis/logging/v2"
	"google.golang.org/grpc"
)

type googleCloudTestCase struct {
	name           string
	config         *GoogleCloudOutputConfig
	input          *entry.Entry
	expectedOutput *logpb.WriteLogEntriesRequest
}

func googleCloudBasicConfig() *GoogleCloudOutputConfig {
	cfg := NewGoogleCloudOutputConfig("test_id")
	cfg.ProjectID = "test_project_id"
	bufferCfg := buffer.NewMemoryBufferConfig()
	bufferCfg.MaxChunkDelay = helper.NewDuration(50 * time.Millisecond)
	cfg.BufferConfig = buffer.Config{Builder: bufferCfg}
	return cfg
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
				Body: map[string]interface{}{
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
				f := entry.NewBodyField("log_name")
				c.LogNameField = &f
				return c
			}(),
			&entry.Entry{
				Timestamp: now,
				Body: map[string]interface{}{
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
			"Attributes",
			func() *GoogleCloudOutputConfig {
				return googleCloudBasicConfig()
			}(),
			&entry.Entry{
				Timestamp: now,
				Attributes: map[string]string{
					"attribute1": "value1",
				},
				Body: map[string]interface{}{
					"message": "test message",
				},
			},
			func() *logpb.WriteLogEntriesRequest {
				req := googleCloudBasicWriteEntriesRequest()
				req.Entries = []*logpb.LogEntry{
					{
						Labels: map[string]string{
							"attribute1": "value1",
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
		googleCloudSeverityTestCase(entry.Severity(15), sev.LogSeverity_DEBUG),
		googleCloudSeverityTestCase(entry.Trace, sev.LogSeverity_DEBUG),
		googleCloudSeverityTestCase(entry.Severity(5), sev.LogSeverity_DEBUG),
		googleCloudSeverityTestCase(entry.Default, sev.LogSeverity_DEFAULT),
		{
			"TraceAndSpanFields",
			func() *GoogleCloudOutputConfig {
				c := googleCloudBasicConfig()
				traceField := entry.NewBodyField("trace")
				spanIDField := entry.NewBodyField("span_id")
				c.TraceField = &traceField
				c.SpanIDField = &spanIDField
				return c
			}(),
			&entry.Entry{
				Timestamp: now,
				Body: map[string]interface{}{
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
			ops, err := tc.config.Build(buildContext)
			op := ops[0]
			require.NoError(t, err)

			conn, received, stop, err := startServer()
			require.NoError(t, err)
			defer stop()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			client, err := vkit.NewClient(ctx, option.WithGRPCConn(conn))
			require.NoError(t, err)
			op.(*GoogleCloudOutput).client = client
			op.(*GoogleCloudOutput).startFlushing()
			defer op.Stop()

			err = op.Process(context.Background(), tc.input)
			require.NoError(t, err)

			select {
			case req := <-received:
				// Apparently there is occasionally an infinite loop in req.stat
				// and testify freezes up trying to infinitely unpack it
				// So instead, just compare the meaningful portions
				require.Equal(t, tc.expectedOutput.LogName, req.LogName)
				require.Equal(t, tc.expectedOutput.Labels, req.Labels)
				require.Equal(t, tc.expectedOutput.Resource, req.Resource)
				require.Equal(t, tc.expectedOutput.Entries, req.Entries)
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
			Body: map[string]interface{}{
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

type googleCloudProtobufTest struct {
	name   string
	record interface{}
}

func (g *googleCloudProtobufTest) Run(t *testing.T) {
	t.Run(g.name, func(t *testing.T) {
		e := &logpb.LogEntry{}
		err := setPayload(e, g.record)
		require.NoError(t, err)
	})
}

func TestGoogleCloudSetPayload(t *testing.T) {
	cases := []googleCloudProtobufTest{
		{
			"string",
			"test",
		},
		{
			"[]byte",
			[]byte("test"),
		},
		{
			"map[string]string",
			map[string]string{"test": "val"},
		},
		{
			"Nested_[]string",
			map[string]interface{}{
				"sub": []string{"1", "2"},
			},
		},
		{
			"Nested_[]int",
			map[string]interface{}{
				"sub": []int{1, 2},
			},
		},
		{
			"Nested_uint32",
			map[string]interface{}{
				"sub": uint32(32),
			},
		},
		{
			"DeepNested_map",
			map[string]interface{}{
				"0": map[string]map[string]map[string]string{
					"1": {"2": {"3": "test"}},
				},
			},
		},
		{
			"DeepNested_slice",
			map[string]interface{}{
				"0": [][][]string{{{"0", "1"}}},
			},
		},
		{
			"AnonymousStruct",
			map[string]interface{}{
				"0": struct{ Field string }{Field: "test"},
			},
		},
		{
			"NamedStruct",
			map[string]interface{}{
				"0": time.Now(),
			},
		},
	}

	for _, tc := range cases {
		tc.Run(t)
	}
}

// Adapted from https://github.com/googleapis/google-cloud-go/blob/master/internal/testutil/server.go
type loggingHandler struct {
	logpb.LoggingServiceV2Server

	received chan *logpb.WriteLogEntriesRequest
}

func (h *loggingHandler) WriteLogEntries(_ context.Context, req *logpb.WriteLogEntriesRequest) (*logpb.WriteLogEntriesResponse, error) {
	h.received <- req
	return &logpb.WriteLogEntriesResponse{}, nil
}

func startServer() (*grpc.ClientConn, chan *logpb.WriteLogEntriesRequest, func(), error) {
	received := make(chan *logpb.WriteLogEntriesRequest, 10)
	serv := grpc.NewServer()
	logpb.RegisterLoggingServiceV2Server(serv, &loggingHandler{
		received: received,
	})

	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return nil, nil, nil, err
	}
	go serv.Serve(lis)

	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
	if err != nil {
		return nil, nil, nil, err
	}

	return conn, received, serv.Stop, nil
}

type googleCloudOutputBenchmark struct {
	name      string
	entry     *entry.Entry
	configMod func(*GoogleCloudOutputConfig)
}

func (g *googleCloudOutputBenchmark) Run(b *testing.B) {
	conn, received, stop, err := startServer()
	require.NoError(b, err)
	defer stop()

	client, err := vkit.NewClient(context.Background(), option.WithGRPCConn(conn))
	require.NoError(b, err)

	cfg := NewGoogleCloudOutputConfig(g.name)
	cfg.ProjectID = "test_project_id"
	if g.configMod != nil {
		g.configMod(cfg)
	}
	ops, err := cfg.Build(testutil.NewBuildContext(b))
	require.NoError(b, err)
	op := ops[0]
	op.(*GoogleCloudOutput).client = client
	op.(*GoogleCloudOutput).timeout = 30 * time.Second
	defer op.(*GoogleCloudOutput).flusher.Stop()

	b.ResetTimer()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < b.N; i++ {
			op.Process(context.Background(), g.entry)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		i := 0
		for i < b.N {
			req := <-received
			i += len(req.Entries)
		}
	}()

	wg.Wait()
	err = op.Stop()
	require.NoError(b, err)
}

func BenchmarkGoogleCloudOutput(b *testing.B) {
	t := time.Date(2007, 01, 01, 10, 15, 32, 0, time.UTC)
	cases := []googleCloudOutputBenchmark{
		{
			"Simple",
			&entry.Entry{
				Timestamp: t,
				Body:      "test",
			},
			nil,
		},
		{
			"MapBody",
			&entry.Entry{
				Timestamp: t,
				Body:      mapOfSize(1, 0),
			},
			nil,
		},
		{
			"LargeMapBody",
			&entry.Entry{
				Timestamp: t,
				Body:      mapOfSize(30, 0),
			},
			nil,
		},
		{
			"DeepMapBody",
			&entry.Entry{
				Timestamp: t,
				Body:      mapOfSize(1, 10),
			},
			nil,
		},
		{
			"Labels",
			&entry.Entry{
				Timestamp: t,
				Body:      "test",
				Attributes: map[string]string{
					"test": "val",
				},
			},
			nil,
		},
		{
			"NoCompression",
			&entry.Entry{
				Timestamp: t,
				Body:      "test",
			},
			func(cfg *GoogleCloudOutputConfig) {
				cfg.UseCompression = false
			},
		},
	}

	for _, tc := range cases {
		b.Run(tc.name, tc.Run)
	}
}

func mapOfSize(keys, depth int) map[string]interface{} {
	m := make(map[string]interface{})
	for i := 0; i < keys; i++ {
		if depth == 0 {
			m["k"+strconv.Itoa(i)] = "v" + strconv.Itoa(i)
		} else {
			m["k"+strconv.Itoa(i)] = mapOfSize(keys, depth-1)
		}
	}
	return m
}

func TestSplittingSender(t *testing.T) {
	cases := []struct {
		name           string
		expectedSplits int
		entries        []*entry.Entry
	}{
		{
			name:           "single_small",
			expectedSplits: 0,
			entries: []*entry.Entry{
				sized(10),
			},
		},
		{
			name:           "total_small",
			expectedSplits: 0,
			entries: []*entry.Entry{
				sized(20),
				sized(20),
			},
		},
		{
			name:           "total_too_large",
			expectedSplits: 1, // split the two apart
			entries: []*entry.Entry{
				sized(51),
				sized(51),
			},
		},
		{
			name:           "many_splits",
			expectedSplits: 8, // n-1 splits
			entries: []*entry.Entry{
				sized(99),
				sized(99),
				sized(99),
				sized(99),
				sized(99),
				sized(99),
				sized(99),
				sized(99),
				sized(99),
			},
		},
		{
			name:           "single_too_large",
			expectedSplits: 1, // replace with error message
			entries: []*entry.Entry{
				sized(101),
			},
		},
		{
			name:           "all_in_one",
			expectedSplits: 3,
			entries: []*entry.Entry{
				// these two should go together after one split
				sized(33),
				sized(66),

				// these two will have to be split again
				sized(99),
				sized(101), // still fails, replace with error message

			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock := mockSender{maxSize: 100}
			split := splittingSender{&mock}

			clearer := newMockClearer(len(tc.entries))
			err := split.Send(context.Background(), clearer, tc.entries, 0)

			require.NoError(t, err, "unexpected error")
			require.Equal(t, tc.expectedSplits, mock.splits, "unexpected number of splits")
			require.True(t, clearer.fullyCleared(), "should be fully cleared")
		})
	}
}

func TestSplittingSenderRandomly(t *testing.T) {

	// pregenerate random samples, but deterministically
	rand.Seed(int64(112233445566778899))

	cases := [][]*entry.Entry{}

	for caseNum := 0; caseNum < 10000; caseNum++ {
		numEntries := rand.Intn(50-1) + 10
		entries := make([]*entry.Entry, 0, numEntries)
		for i := 0; i < numEntries; i++ {
			bodyLength := rand.Intn(101-1) + 1
			entries = append(entries, sized(bodyLength))
		}
		cases = append(cases, entries)
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			mock := mockSender{maxSize: 100}
			split := splittingSender{&mock}

			clearer := newMockClearer(len(tc))
			err := split.Send(context.Background(), clearer, tc, 0)

			require.NoError(t, err, "unexpected error")
			require.True(t, clearer.fullyCleared(), "should be fully cleared")
		})
	}
}

const charset = "abcdefghijklmnopqrstuvwxyz"

func sized(i int) *entry.Entry {
	ent := entry.New()
	b := make([]byte, i)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	ent.Body = string(b)
	return ent
}

type mockSender struct {
	maxSize int
	splits  int
}

func (s *mockSender) Debugf(template string, args ...interface{}) {
	return
}

func (s *mockSender) Debugw(template string, args ...interface{}) {
	return
}

func (s *mockSender) Send(_ context.Context, entries []*entry.Entry) error {
	totalSize := 0
	for _, ent := range entries {
		s, ok := ent.Body.(string)
		if !ok {
			panic("unexpected value in test")
		}
		totalSize += len(s)
	}

	if totalSize > s.maxSize {
		s.splits++

		// it's important that this is short enough
		// because it becomes the record if a single
		// entry is too large
		return fmt.Errorf("too big")
	}

	return nil
}

func (s *mockSender) IsTooLargeError(err error) bool {
	return err.Error() == "too big"
}

type mockClearer struct {
	cleared []bool
}

func newMockClearer(l int) mockClearer {
	return mockClearer{make([]bool, l)}
}

func (c mockClearer) MarkAllAsFlushed() error {
	for i := 0; i < len(c.cleared); i++ {
		if c.cleared[i] {
			return fmt.Errorf("already cleared: %d", i)
		}
		c.cleared[i] = true
	}
	return nil
}

func (c mockClearer) MarkRangeAsFlushed(start, end uint) error {
	if start < uint(0) {
		return fmt.Errorf("Clearer index out of bounds: %d", start)
	}

	if end > uint(len(c.cleared)) {
		return fmt.Errorf("Clearer index out of bounds: %d", end)
	}

	for i := start; i < end; i++ {
		if c.cleared[i] {
			return fmt.Errorf("already cleared: %d", i)
		}
		c.cleared[i] = true
	}
	return nil
}

func (c mockClearer) fullyCleared() bool {
	for i := 0; i < len(c.cleared); i++ {
		if !c.cleared[i] {
			return false
		}
	}
	return true
}
