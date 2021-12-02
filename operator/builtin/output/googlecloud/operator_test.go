package googlecloud

import (
	"context"
	"errors"
	"testing"

	"github.com/googleapis/gax-go/v2"
	"github.com/observiq/stanza/v2/testutil"
	"github.com/open-telemetry/opentelemetry-log-collection/entry"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/option"
	"google.golang.org/genproto/googleapis/logging/v2"
)

func TestStartClientFailure(t *testing.T) {
	operator := createTestOperator(t)
	operator.buildClient = func(ctx context.Context, opts ...option.ClientOption) (Client, error) {
		return nil, errors.New("client failure")
	}
	persister := &testutil.MockPersister{}

	err := operator.Start(persister)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to create client")
}

func TestStartTestConnectionFailure(t *testing.T) {
	client := &MockClient{}
	client.On("WriteLogEntries", mock.Anything, mock.Anything).Return(nil, errors.New("failed response")).Once()

	operator := createTestOperator(t)
	operator.buildClient = func(ctx context.Context, opts ...option.ClientOption) (Client, error) {
		return client, nil
	}
	persister := &testutil.MockPersister{}

	err := operator.Start(persister)
	require.Error(t, err)
	require.Contains(t, err.Error(), "test connection failed")
}

func TestStartSuccess(t *testing.T) {
	client := &MockClient{}
	client.On("WriteLogEntries", mock.Anything, mock.Anything).Return(nil, nil)
	client.On("Close").Return(nil)

	operator := createTestOperator(t)
	operator.buildClient = func(ctx context.Context, opts ...option.ClientOption) (Client, error) {
		return client, nil
	}
	persister := &testutil.MockPersister{}

	err := operator.Start(persister)
	require.NoError(t, err)

	err = operator.Stop()
	require.NoError(t, err)
}

func TestFlusher(t *testing.T) {
	t.Skip("skipping test until buffer is ready")
	writeChan := make(chan interface{})
	writeFunc := func(args mock.Arguments) { writeChan <- args[1] }

	client := &MockClient{}
	client.On("WriteLogEntries", mock.Anything, mock.Anything).Return(nil, nil).Once()
	client.On("WriteLogEntries", mock.Anything, mock.Anything).Return(nil, errors.New("failed to send")).Once()
	client.On("WriteLogEntries", mock.Anything, mock.Anything).Run(writeFunc).Return(nil, nil).Once()
	client.On("Close").Return(nil)

	operator := createTestOperator(t)
	operator.buildClient = func(ctx context.Context, opts ...option.ClientOption) (Client, error) {
		return client, nil
	}
	persister := &testutil.MockPersister{}

	err := operator.Start(persister)
	require.NoError(t, err)

	testEntry := &entry.Entry{Body: "test record"}
	operator.Process(context.Background(), testEntry)

	request := <-writeChan
	logRequest, ok := request.(*logging.WriteLogEntriesRequest)
	require.True(t, ok)
	require.Equal(t, logRequest.Entries[0].GetTextPayload(), "test record")

	err = operator.Stop()
	require.NoError(t, err)
}

func TestBufferFailure(t *testing.T) {
	bufferChan := make(chan interface{})
	bufferFunc := func(args mock.Arguments) { close(bufferChan) }

	buffer := &MockBuffer{}
	buffer.On("Add", mock.Anything, mock.Anything).Return(nil, nil)
	buffer.On("Read", mock.Anything).Run(bufferFunc).Return(nil, errors.New("first failure")).Once()
	buffer.On("Read", mock.Anything).Return(nil, errors.New("continued failures"))
	buffer.On("Close").Return(nil)

	client := &MockClient{}
	client.On("WriteLogEntries", mock.Anything, mock.Anything).Return(nil, nil).Once()
	client.On("Close").Return(nil)

	operator := createTestOperator(t)
	operator.buffer = buffer
	operator.buildClient = func(ctx context.Context, opts ...option.ClientOption) (Client, error) {
		return client, nil
	}
	persister := &testutil.MockPersister{}

	err := operator.Start(persister)
	require.NoError(t, err)

	testEntry := &entry.Entry{Body: "test record"}
	operator.Process(context.Background(), testEntry)

	<-bufferChan

	err = operator.Stop()
	require.NoError(t, err)
}

func TestStopMissingClient(t *testing.T) {
	operator := createTestOperator(t)
	err := operator.Stop()
	require.NoError(t, err)
}

func TestStopClientFailure(t *testing.T) {
	client := &MockClient{}
	client.On("Close").Return(errors.New("failure"))

	operator := createTestOperator(t)
	operator.client = client

	err := operator.Stop()
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to close client")
}

func TestStopBufferFailure(t *testing.T) {
	buffer := &MockBuffer{}
	buffer.On("Close").Return(errors.New("failure"))

	operator := createTestOperator(t)
	operator.buffer = buffer

	err := operator.Stop()
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to close buffer")
}

// Client is a mock type for the Client interface
type MockClient struct {
	mock.Mock
}

// Close provides a mock close function
func (_m *MockClient) Close() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// WriteLogEntries provides a mock function with given fields: ctx, req, opts
func (_m *MockClient) WriteLogEntries(ctx context.Context, req *logging.WriteLogEntriesRequest, opts ...gax.CallOption) (*logging.WriteLogEntriesResponse, error) {
	_va := make([]interface{}, len(opts))
	for _i := range opts {
		_va[_i] = opts[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, req)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 *logging.WriteLogEntriesResponse
	if rf, ok := ret.Get(0).(func(context.Context, *logging.WriteLogEntriesRequest, ...gax.CallOption) *logging.WriteLogEntriesResponse); ok {
		r0 = rf(ctx, req, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*logging.WriteLogEntriesResponse)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *logging.WriteLogEntriesRequest, ...gax.CallOption) error); ok {
		r1 = rf(ctx, req, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockBuffer is an autogenerated mock type for the Buffer type
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
func (_m *MockBuffer) Close() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Drain provides a mock function with given fields: _a0
func (_m *MockBuffer) Drain(_a0 context.Context) ([]*entry.Entry, error) {
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

// createTestOperator creates a basic test operator
func createTestOperator(t *testing.T) *GoogleCloudOutput {
	config := NewGoogleCloudOutputConfig("test")
	config.Credentials = `{"type": "service_account", "project_id": "test"}`

	buildContext := testutil.NewBuildContext(t)
	operators, err := config.Build(buildContext)

	require.NoError(t, err)
	require.Len(t, operators, 1)

	googleOperator, ok := operators[0].(*GoogleCloudOutput)
	require.True(t, ok)

	return googleOperator
}
