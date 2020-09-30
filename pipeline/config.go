package pipeline

import (
	"github.com/observiq/stanza/operator"
)

// Config is the configuration of a pipeline.
type Config []operator.Builder

// BuildPipeline will build a pipeline from the config.
func (c Config) BuildPipeline(context operator.BuildContext) (*DirectedPipeline, error) {
	// TODO validate
	// TODO default output

	operators := make([]operator.Operator, 0, len(c))
	for _, builder := range c {
		op, err := builder.Build(context)
		if err != nil {
			return nil, err
		}
		operators = append(operators, op)
	}

	return NewDirectedPipeline(operators)
}
