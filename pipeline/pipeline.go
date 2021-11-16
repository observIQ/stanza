//go:generate mockery --name=^(Pipeline)$ --output=../testutil --outpkg=testutil --case=snake

package pipeline

import "github.com/observiq/stanza/v2/operator"

// Pipeline is a collection of connected operators that exchange entries
type Pipeline interface {
	Start() error
	Stop() error
	Operators() []operator.Operator
	Render() ([]byte, error)
}
