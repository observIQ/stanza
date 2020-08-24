package testutil

import (
	context "context"
	"testing"

	entry "github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator"
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

type FakeOutput struct {
	Received chan *entry.Entry
	*zap.SugaredLogger
}

func NewFakeOutput(t testing.TB) *FakeOutput {
	return &FakeOutput{
		Received:      make(chan *entry.Entry, 100),
		SugaredLogger: zaptest.NewLogger(t).Sugar(),
	}
}

func (f *FakeOutput) CanOutput() bool                              { return false }
func (f *FakeOutput) CanProcess() bool                             { return true }
func (f *FakeOutput) ID() string                                   { return "fake" }
func (f *FakeOutput) Logger() *zap.SugaredLogger                   { return f.SugaredLogger }
func (f *FakeOutput) Outputs() []operator.Operator                 { return nil }
func (f *FakeOutput) SetOutputs(outputs []operator.Operator) error { return nil }
func (f *FakeOutput) AddOutput(output operator.Operator) error     { return nil }
func (f *FakeOutput) Start() error                                 { return nil }
func (f *FakeOutput) Stop() error                                  { return nil }
func (f *FakeOutput) Type() string                                 { return "fake_output" }

func (f *FakeOutput) Process(ctx context.Context, entry *entry.Entry) error {
	f.Received <- entry
	return nil
}
