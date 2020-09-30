package pipeline

import (
	"github.com/observiq/stanza/operator"
)

// Config is the configuration of a pipeline.
type Config []operator.Config

// BuildPipeline will build a pipeline from the config.
func (c Config) BuildPipeline(bc operator.BuildContext) (*DirectedPipeline, error) {
	operators := make([]operator.Operator, 0, len(c))
	for i, builder := range c {
		nbc := getBuildContextWithDefaultOutput(c, i, bc)
		op, err := builder.Build(nbc)
		if err != nil {
			return nil, err
		}
		operators = append(operators, op)
	}

	return NewDirectedPipeline(operators)
}

func getBuildContextWithDefaultOutput(configs []operator.Config, i int, bc operator.BuildContext) operator.BuildContext {
	if i+1 >= len(configs) {
		return bc
	}

	id := configs[i+1].ID()
	id = bc.PrependNamespace(id)
	return bc.WithDefaultOutputIDs([]string{id})
}
