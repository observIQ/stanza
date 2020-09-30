package plugin

import (
	"context"
	"fmt"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/helper"
	"github.com/observiq/stanza/pipeline"
)

var _ operator.Operator = (*PluginOperator)(nil)

type PluginOperator struct {
	helper.BasicOperator
	Pipeline   pipeline.Pipeline
	Entrypoint operator.Operator
}

func (p *PluginOperator) Start() error {
	return p.Pipeline.Start()
}

func (p *PluginOperator) Stop() error {
	return p.Pipeline.Stop()
}

func (p *PluginOperator) CanOutput() bool { return true }
func (p *PluginOperator) Outputs() []operator.Operator {
	// Create a set of unique outputs
	uniqueOutputs := map[operator.Operator]struct{}{}
	for _, operator := range p.Pipeline.Operators() {
		for _, output := range operator.Outputs() {
			uniqueOutputs[output] = struct{}{}
		}
	}

	// Convert the set to an array
	outputs := make([]operator.Operator, 0, len(uniqueOutputs))
	for operator := range uniqueOutputs {
		outputs = append(outputs, operator)
	}

	return outputs
}

func (p *PluginOperator) SetOutputs(operators []operator.Operator) error {
	// Each operator in the plugin's pipeline can see the other operators inside its pipeline,
	// and all the operators that are visible to the plugin operator itself
	allVisibleOperators := append(operators, p.Pipeline.Operators()...)
	for _, operator := range p.Pipeline.Operators() {
		if err := operator.SetOutputs(allVisibleOperators); err != nil {
			return err
		}
	}
	return nil
}

func (p *PluginOperator) CanProcess() bool {
	return p.Entrypoint != nil
}

func (p *PluginOperator) Process(ctx context.Context, entry *entry.Entry) error {
	if p.Entrypoint == nil {
		return fmt.Errorf("plugin has no entrypoint, so cannot process entries")
	}
	return p.Entrypoint.Process(ctx, entry)
}
