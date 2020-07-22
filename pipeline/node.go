package pipeline

import (
	"hash/fnv"

	"github.com/observiq/carbon/plugin"
)

// OperatorNode is a basic node that represents a plugin in a pipeline.
type OperatorNode struct {
	plugin    plugin.Operator
	id        int64
	outputIDs map[string]int64
}

// Operator returns the plugin of the node.
func (b OperatorNode) Operator() plugin.Operator {
	return b.plugin
}

// ID returns the node id.
func (b OperatorNode) ID() int64 {
	return b.id
}

// DOTID returns the id used to represent this node in a dot graph.
func (b OperatorNode) DOTID() string {
	return b.plugin.ID()
}

// OutputIDs returns a map of output plugin ids to node ids.
func (b OperatorNode) OutputIDs() map[string]int64 {
	return b.outputIDs
}

// createOperatorNode will create a plugin node.
func createOperatorNode(plugin plugin.Operator) OperatorNode {
	id := createNodeID(plugin.ID())
	outputIDs := make(map[string]int64)
	if plugin.CanOutput() {
		for _, output := range plugin.Outputs() {
			outputIDs[output.ID()] = createNodeID(output.ID())
		}
	}
	return OperatorNode{plugin, id, outputIDs}
}

// createNodeID generates a node id from a plugin id.
func createNodeID(pluginID string) int64 {
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(pluginID))
	return int64(hash.Sum64())
}
