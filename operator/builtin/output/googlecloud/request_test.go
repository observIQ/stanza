package googlecloud

import (
	"errors"
	"testing"

	"github.com/observiq/stanza/entry"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/genproto/googleapis/logging/v2"
)

func TestBuildRequest(t *testing.T) {
	entryOne := &entry.Entry{Record: "2 bytes"}
	entryTwo := &entry.Entry{Record: "3 bytes"}
	entryThree := &entry.Entry{Record: "error"}
	entryFour := &entry.Entry{Record: "5 bytes"}

	resultOne := &logging.LogEntry{InsertId: "2 bytes"}
	resultTwo := &logging.LogEntry{InsertId: "3 bytes"}
	resultFour := &logging.LogEntry{InsertId: "5 bytes"}

	entryBuilder := &MockEntryBuilder{}
	entryBuilder.On("Build", entryOne).Return(resultOne, 2, nil)
	entryBuilder.On("Build", entryTwo).Return(resultTwo, 3, nil)
	entryBuilder.On("Build", entryThree).Return(nil, 0, errors.New("error"))
	entryBuilder.On("Build", entryFour).Return(resultFour, 5, nil)

	entries := []*entry.Entry{entryOne, entryTwo, entryThree, entryFour}
	requestBuilder := GoogleRequestBuilder{
		MaxRequestSize: 5,
		ProjectID:      "test_project",
		EntryBuilder:   entryBuilder,
		SugaredLogger:  zap.NewNop().Sugar(),
	}

	requests := requestBuilder.Build(entries)
	require.Len(t, requests, 2)

	require.Len(t, requests[0].Entries, 2)
	require.Equal(t, requests[0].Entries, []*logging.LogEntry{resultOne, resultTwo})
	require.Equal(t, requests[1].Entries, []*logging.LogEntry{resultFour})
}

// MockEntryBuilder is a mock for the EntryBuilder interface
type MockEntryBuilder struct {
	mock.Mock
}

// Build mocks the build function
func (_m *MockEntryBuilder) Build(_a0 *entry.Entry) (*logging.LogEntry, int, error) {
	ret := _m.Called(_a0)

	var r0 *logging.LogEntry
	if rf, ok := ret.Get(0).(func(*entry.Entry) *logging.LogEntry); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*logging.LogEntry)
		}
	}

	var r1 int
	if rf, ok := ret.Get(1).(func(*entry.Entry) int); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Get(1).(int)
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(*entry.Entry) error); ok {
		r2 = rf(_a0)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
