package googlecloud

import (
	"errors"
	"fmt"
	"testing"

	"github.com/observiq/stanza/entry"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/genproto/googleapis/logging/v2"
)

func BenchmarkBuildRequest(b *testing.B) {
	entryBuilder := &GoogleEntryBuilder{
		MaxEntrySize: defaultMaxEntrySize,
		ProjectID:    "project",
	}
	entries := []*entry.Entry{}

	for i := 0; i < 1000; i++ {
		entry, _ := createEntry(i)
		entries = append(entries, entry)
	}

	requestBuilder := GoogleRequestBuilder{
		MaxRequestSize: 10000,
		ProjectID:      "test_project",
		EntryBuilder:   entryBuilder,
		SugaredLogger:  zap.NewNop().Sugar(),
	}

	requests := requestBuilder.Build(entries)
	require.Len(b, requests, 3)
}

func TestBuildRequest(t *testing.T) {
	entryOne := &entry.Entry{Record: "request 1"}
	entryTwo := &entry.Entry{Record: "request 2"}
	entryThree := &entry.Entry{Record: "request 3"}
	entryFour := &entry.Entry{Record: "request 4"}

	resultOne := &logging.LogEntry{Payload: &logging.LogEntry_TextPayload{TextPayload: "request 1"}}
	resultTwo := &logging.LogEntry{Payload: &logging.LogEntry_TextPayload{TextPayload: "request 2"}}
	resultFour := &logging.LogEntry{Payload: &logging.LogEntry_TextPayload{TextPayload: "request 3"}}

	entryBuilder := &MockEntryBuilder{}
	entryBuilder.On("Build", entryOne).Return(resultOne, nil)
	entryBuilder.On("Build", entryTwo).Return(resultTwo, nil)
	entryBuilder.On("Build", entryThree).Return(nil, errors.New("error"))
	entryBuilder.On("Build", entryFour).Return(resultFour, nil)

	entries := []*entry.Entry{entryOne, entryTwo, entryThree, entryFour}
	requestBuilder := GoogleRequestBuilder{
		MaxRequestSize: 100,
		ProjectID:      "test_project",
		EntryBuilder:   entryBuilder,
		SugaredLogger:  zap.NewNop().Sugar(),
	}

	requests := requestBuilder.Build(entries)
	require.Len(t, requests, 2)

	require.Len(t, requests[0].Entries, 1)
	require.Len(t, requests[1].Entries, 2)
	require.Equal(t, requests[0].Entries, []*logging.LogEntry{resultOne})
	require.Equal(t, requests[1].Entries, []*logging.LogEntry{resultTwo, resultFour})
}

func TestImpossibleEntry(t *testing.T) {
	entryOne := &entry.Entry{Record: "Test Request"}
	resultOne := &logging.LogEntry{Payload: &logging.LogEntry_TextPayload{TextPayload: "Test Request"}}

	entryBuilder := &MockEntryBuilder{}
	entryBuilder.On("Build", entryOne).Return(resultOne, nil)

	entries := []*entry.Entry{entryOne}
	requestBuilder := GoogleRequestBuilder{
		MaxRequestSize: 1,
		ProjectID:      "test_project",
		EntryBuilder:   entryBuilder,
		SugaredLogger:  zap.NewNop().Sugar(),
	}

	requests := requestBuilder.Build(entries)
	require.Len(t, requests, 0)
}

// MockEntryBuilder is a mock for the EntryBuilder interface
type MockEntryBuilder struct {
	mock.Mock
}

// Build provides a mock function with given fields: _a0
func (_m *MockEntryBuilder) Build(_a0 *entry.Entry) (*logging.LogEntry, error) {
	ret := _m.Called(_a0)

	var r0 *logging.LogEntry
	if rf, ok := ret.Get(0).(func(*entry.Entry) *logging.LogEntry); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*logging.LogEntry)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*entry.Entry) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

func createEntry(num int) (*entry.Entry, *logging.LogEntry) {
	entry := &entry.Entry{Record: fmt.Sprintf("request %d", num)}
	result := &logging.LogEntry{Payload: &logging.LogEntry_TextPayload{TextPayload: fmt.Sprintf("request %d", num)}}
	return entry, result
}
