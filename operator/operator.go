//go:generate mockery --name=^(Operator)$ --output=../testutil --outpkg=testutil --case=snake

package operator

import (
	"context"

	"github.com/observiq/stanza/entry"
	"go.uber.org/zap"
)

// Operator is a log monitoring component.
type Operator interface {
	// ID returns the id of the operator.
	ID() string
	// Type returns the type of the operator.
	Type() string

	// Start will start the operator.
	Start() error
	// Stop will stop the operator.
	Stop() error

	// CanOutput indicates if the operator will output entries to other operators.
	CanOutput() bool
	// Outputs returns the list of connected outputs.
	Outputs() []Operator
	// SetOutputs will set the connected outputs.
	SetOutputs([]Operator) error

	// CanProcess indicates if the operator will process entries from other operators.
	CanProcess() bool
	// Process will process an entry from an operator.
	Process(context.Context, *entry.Entry) error
	// Logger returns the operator's logger
	Logger() *zap.SugaredLogger
}
