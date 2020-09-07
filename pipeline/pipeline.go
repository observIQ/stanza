//go:generate mockery -name=^(Pipeline)$ -output=../testutil -outpkg=testutil -case=snake

package pipeline

// Pipeline is a collection of connected operators that exchange entries
type Pipeline interface {
	Start() error
	Stop() error
	Render() ([]byte, error)
	Running() bool
}
