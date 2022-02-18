package testutil

import (
	context "context"
	"testing"
	"time"

	entry "github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator"
	"github.com/stretchr/testify/require"
	zap "go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// NewMockOperator will return a basic operator mock
func NewMockOperator(id string) *Operator {
	mockOutput := &Operator{}
	mockOutput.On("ID").Return(id)
	mockOutput.On("CanProcess").Return(true)
	mockOutput.On("CanOutput").Return(true)
	return mockOutput
}

// FakeOutput is an empty output used primarily for testing
type FakeOutput struct {
	Received chan *entry.Entry
	*zap.SugaredLogger
}

// NewFakeOutput creates a new fake output with default settings
func NewFakeOutput(t testing.TB) *FakeOutput {
	return &FakeOutput{
		Received:      make(chan *entry.Entry, 100),
		SugaredLogger: zaptest.NewLogger(t).Sugar(),
	}
}

// CanOutput always returns false for a fake output
func (f *FakeOutput) CanOutput() bool { return false }

// CanProcess always returns true for a fake output
func (f *FakeOutput) CanProcess() bool { return true }

// ID always returns `fake` as the ID of a fake output operator
func (f *FakeOutput) ID() string { return "$.fake" }

// Logger returns the logger of a fake output
func (f *FakeOutput) Logger() *zap.SugaredLogger { return f.SugaredLogger }

// Outputs always returns nil for a fake output
func (f *FakeOutput) Outputs() []operator.Operator { return nil }

// SetOutputs immediately returns nil for a fake output
func (f *FakeOutput) SetOutputs(_ []operator.Operator) error { return nil }

// Start immediately returns nil for a fake output
func (f *FakeOutput) Start() error { return nil }

// Stop immediately returns nil for a fake output
func (f *FakeOutput) Stop() error { return nil }

// Type always return `fake_output` for a fake output
func (f *FakeOutput) Type() string { return "fake_output" }

// Process will place all incoming entries on the Received channel of a fake output
func (f *FakeOutput) Process(_ context.Context, entry *entry.Entry) error {
	f.Received <- entry
	return nil
}

// ExpectRecord expects that a record will be received by the fake operator within a second
// and that it is equal to the given record
func (f *FakeOutput) ExpectRecord(t testing.TB, record interface{}) {
	select {
	case e := <-f.Received:
		require.Equal(t, record, e.Record)
	case <-time.After(time.Second):
		require.FailNow(t, "Timed out waiting for entry")
	}
}

// ExpectEntry expects that an entry will be received by the fake operator within a second
// and that it is equal to the given record
func (f *FakeOutput) ExpectEntry(t testing.TB, expected *entry.Entry) {
	select {
	case e := <-f.Received:
		require.Equal(t, expected, e)
	case <-time.After(time.Second):
		require.FailNow(t, "Timed out waiting for entry")
	}
}

// ExpectNoEntry expects that no entry will be received within the specified time
func (f *FakeOutput) ExpectNoEntry(t testing.TB, timeout time.Duration) {
	select {
	case <-f.Received:
		require.FailNow(t, "Should not have received entry")
	case <-time.After(timeout):
		return
	}
}
