package newrelic

import (
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/observiq/stanza/v2/operator/buffer"
	"github.com/observiq/stanza/v2/operator/flusher"
	"github.com/observiq/stanza/v2/testutil"
	"github.com/open-telemetry/opentelemetry-log-collection/entry"
	"github.com/open-telemetry/opentelemetry-log-collection/operator/helper"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewRelicConfigBuild(t *testing.T) {
	t.Run("OutputConfigError", func(t *testing.T) {
		cfg := NewNewRelicOutputConfig("test")
		cfg.OperatorType = ""
		_, err := cfg.Build(testutil.NewBuildContext(t))
		require.Error(t, err)
		require.Contains(t, err.Error(), "missing required `type` field")
	})

	t.Run("MissingKey", func(t *testing.T) {
		cfg := NewNewRelicOutputConfig("test")
		_, err := cfg.Build(testutil.NewBuildContext(t))
		require.Error(t, err)
		require.Equal(t, err.Error(), "one of 'api_key' or 'license_key' is required")
	})

	t.Run("InvalidURL", func(t *testing.T) {
		cfg := NewNewRelicOutputConfig("test")
		cfg.LicenseKey = "testkey"
		cfg.BaseURI = `%^&*($@)`
		_, err := cfg.Build(testutil.NewBuildContext(t))
		require.Error(t, err)
		require.Contains(t, err.Error(), "is not a valid URL")
	})
}

type testCase struct {
	name     string
	cfgMod   func(*NewRelicOutputConfig)
	input    []*entry.Entry
	expected string
}

func TestNewRelicOutput(t *testing.T) {
	cases := []testCase{
		{
			"Simple",
			nil,
			[]*entry.Entry{{
				Timestamp: time.Date(2016, 10, 10, 8, 58, 52, 0, time.UTC),
				Body:      "test",
			}},
			`[{"common":{"attributes":{"plugin":{"type":"stanza","version":"unknown"}}},"logs":[{"timestamp":1476089932000,"attributes":{"attributes":null,"body":"test","resource":null,"severity":"default"},"message":"test"}]}]` + "\n",
		},
		{
			"Multi",
			nil,
			[]*entry.Entry{{
				Timestamp: time.Date(2016, 10, 10, 8, 58, 52, 0, time.UTC),
				Body:      "test1",
			}, {
				Timestamp: time.Date(2016, 10, 10, 8, 58, 52, 0, time.UTC),
				Body:      "test2",
			}},
			`[{"common":{"attributes":{"plugin":{"type":"stanza","version":"unknown"}}},"logs":[{"timestamp":1476089932000,"attributes":{"attributes":null,"body":"test1","resource":null,"severity":"default"},"message":"test1"},{"timestamp":1476089932000,"attributes":{"attributes":null,"body":"test2","resource":null,"severity":"default"},"message":"test2"}]}]` + "\n",
		},
		{
			"CustomMessage",
			func(cfg *NewRelicOutputConfig) {
				cfg.MessageField = entry.NewBodyField("log")
			},
			[]*entry.Entry{{
				Timestamp: time.Date(2016, 10, 10, 8, 58, 52, 0, time.UTC),
				Body: map[string]interface{}{
					"log":     "testlog",
					"message": "testmessage",
				},
			}},
			`[{"common":{"attributes":{"plugin":{"type":"stanza","version":"unknown"}}},"logs":[{"timestamp":1476089932000,"attributes":{"attributes":null,"body":{"log":"testlog","message":"testmessage"},"resource":null,"severity":"default"},"message":"testlog"}]}]` + "\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			persiter := &testutil.MockPersister{}

			op, ln := createTestOperator(t, tc)
			defer ln.stop()

			require.NoError(t, op.Start(persiter))
			defer op.Stop()

			for _, entry := range tc.input {
				require.NoError(t, op.Process(context.Background(), entry))
			}

			expectTestConnection(t, ln)
			expectRequestBody(t, ln, tc.expected)

		})
	}

	t.Run("FailedTestConnection", func(t *testing.T) {
		persiter := &testutil.MockPersister{}
		cfg := NewNewRelicOutputConfig("test")
		cfg.BaseURI = "http://localhost/log/v1"
		cfg.APIKey = "testkey"

		ops, err := cfg.Build(testutil.NewBuildContext(t))
		require.NoError(t, err)
		op := ops[0]
		err = op.Start(persiter)
		require.Error(t, err)
	})
}

func createTestOperator(t *testing.T, tc testCase) (*NewRelicOutput, *listener) {
	ln := newListener()
	addr, err := ln.start()
	require.NoError(t, err)

	config := NewNewRelicOutputConfig("test")
	config.BufferConfig = buffer.Config{
		Builder: func() buffer.Builder {
			cfg := buffer.NewMemoryBufferConfig()
			cfg.MaxChunkDelay = helper.NewDuration(50 * time.Millisecond)
			return cfg
		}(),
	}

	config.BaseURI = fmt.Sprintf("http://%s/log/v1", addr)
	config.APIKey = "testkey"
	if tc.cfgMod != nil {
		tc.cfgMod(config)
	}

	buildContext := testutil.NewBuildContext(t)
	operators, err := config.Build(buildContext)

	require.NoError(t, err)
	require.Len(t, operators, 1)

	nro, ok := operators[0].(*NewRelicOutput)
	require.True(t, ok)

	return nro, ln
}

func expectTestConnection(t *testing.T, ln *listener) {
	testConnection := `[{"common":{"attributes":{"plugin":{"type":"stanza","version":"unknown"}}},"logs":[]}]` + "\n"
	expectRequestBody(t, ln, testConnection)
}

func expectRequestBody(t *testing.T, ln *listener, expected string) {
	select {
	case body := <-ln.requestBodies:
		require.Equal(t, expected, string(body))
	case <-time.After(time.Second):
		require.FailNow(t, "Timed out waiting for test connection")
	}
}

type listener struct {
	server        *http.Server
	requestBodies chan []byte
}

func newListener() *listener {
	requests := make(chan []byte, 100)

	mux := http.NewServeMux()
	mux.HandleFunc("/", handle(requests))

	return &listener{
		server: &http.Server{
			Handler: mux,
		},
		requestBodies: requests,
	}
}

func (l *listener) start() (string, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}

	go func() {
		l.server.Serve(ln)
	}()

	return ln.Addr().String(), nil
}

func (l *listener) stop() {
	l.server.Shutdown(context.Background())
}

func handle(ch chan []byte) func(rw http.ResponseWriter, req *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(200)
		rw.Write([]byte(`{}`))

		rd, err := gzip.NewReader(req.Body)
		if err != nil {
			panic(err)
		}
		body, err := ioutil.ReadAll(rd)
		if err != nil {
			panic(err)
		}
		req.Body.Close()

		ch <- body
	}
}

func createTestOp(t *testing.T) (*NewRelicOutput, *MockClient) {
	client := &MockClient{}

	buffer, err := buffer.NewConfig().Build()
	require.NoError(t, err)

	flusher := flusher.NewConfig().Build(zap.NewNop().Sugar())
	ctx, cancel := context.WithCancel(context.Background())

	operator := NewRelicOutput{
		messageField: entry.NewBodyField(),
		buffer:       buffer,
		client:       client,
		flusher:      flusher,
		ctx:          ctx,
		cancel:       cancel,
	}

	return &operator, client
}

func TestCloseSuccessfulFlush(t *testing.T) {
	operator, client := createTestOp(t)

	entry := &entry.Entry{Body: "test message"}
	operator.buffer.Add(context.Background(), entry)

	sendChan := make(chan LogPayload, 1)
	client.On("SendPayload", mock.Anything, mock.Anything).Run(func(args mock.Arguments) { sendChan <- args[1].(LogPayload) }).Return(nil)

	err := operator.Stop()
	require.NoError(t, err)

	payloads := <-sendChan
	require.Len(t, payloads, 1)

	logs := payloads[0].Logs
	require.Len(t, logs, 1)
	require.Equal(t, "test message", logs[0].Message)
}

func TestCloseFailedFlush(t *testing.T) {
	operator, client := createTestOp(t)

	entry := &entry.Entry{Body: "test message"}
	operator.buffer.Add(context.Background(), entry)

	client.On("SendPayload", mock.Anything, mock.Anything).Return(errors.New("failed to flush"))

	err := operator.Stop()
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to flush")
}
func TestBufferCloseFailure(t *testing.T) {
	buffer := &MockBuffer{}
	operator, _ := createTestOp(t)
	operator.buffer = buffer

	buffer.On("Close").Return(nil, errors.New("failed to close buffer"))

	err := operator.Stop()
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to close buffer")
}

func TestBufferReadFailure(t *testing.T) {
	buffer := &MockBuffer{}
	operator, client := createTestOp(t)
	operator.buffer = buffer

	buffer.On("Read", mock.Anything).Return(nil, errors.New("failure")).Once()
	buffer.On("Read", mock.Anything).Return(nil, context.Canceled).Once()
	buffer.On("Close").Return(nil, nil)

	client.On("TestConnection", mock.Anything).Return(nil)

	doneChan := make(chan struct{}, 1)
	go func() {
		defer close(doneChan)
		operator.wg.Wait()
	}()

	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
	select {
	case <-doneChan:
		//feedflusher exited successfully
	case <-timer.C:
		require.Fail(t, "test timed out")
	}
}

// Buffer is an autogenerated mock type for the Buffer type
type MockBuffer struct {
	mock.Mock
}

// Add provides a mock function with given fields: _a0, _a1
func (_m *MockBuffer) Add(_a0 context.Context, _a1 *entry.Entry) error {
	ret := _m.Called(_a0, _a1)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *entry.Entry) error); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Close provides a mock function with given fields:
func (_m *MockBuffer) Close() ([]*entry.Entry, error) {
	ret := _m.Called()

	var r0 []*entry.Entry
	if rf, ok := ret.Get(0).(func() []*entry.Entry); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*entry.Entry)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Read provides a mock function with given fields: _a0
func (_m *MockBuffer) Read(_a0 context.Context) ([]*entry.Entry, error) {
	ret := _m.Called(_a0)

	var r0 []*entry.Entry
	if rf, ok := ret.Get(0).(func(context.Context) []*entry.Entry); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*entry.Entry)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockClient is an autogenerated mock type for the Client type
type MockClient struct {
	mock.Mock
}

// SendPayload provides a mock function with given fields: _a0, _a1
func (_m *MockClient) SendPayload(_a0 context.Context, _a1 LogPayload) error {
	ret := _m.Called(_a0, _a1)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, LogPayload) error); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// TestConnection provides a mock function with given fields: _a0
func (_m *MockClient) TestConnection(_a0 context.Context) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
