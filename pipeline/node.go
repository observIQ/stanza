package pipeline

import (
	"fmt"
	"hash/fnv"

	"github.com/bluemedora/bplogagent/plugin"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/encoding/dot"
)

// PluginNode is a node that represents a plugin.
type PluginNode interface {
	Plugin() plugin.Plugin
	ID() int64
	DOTID() string
	OutputIDs() map[string]int64
}

// createPluginNode will create a plugin node from a plugin.
func createPluginNode(plugin plugin.Plugin) PluginNode {
	if _, ok := plugin.(dot.Subgrapher); ok {
		return createSubgraphNode(plugin)
	}
	return createBasicNode(plugin)
}

// BasicPluginNode is a basic node that represents a plugin in a pipeline.
type BasicPluginNode struct {
	plugin    plugin.Plugin
	id        int64
	outputIDs map[string]int64
}

// Plugin returns the plugin of the node.
func (b BasicPluginNode) Plugin() plugin.Plugin {
	return b.plugin
}

// ID returns the node id.
func (b BasicPluginNode) ID() int64 {
	return b.id
}

// DOTID returns the id used to represent this node in a dot graph.
func (b BasicPluginNode) DOTID() string {
	return fmt.Sprintf("%s: %s", b.plugin.Type(), b.plugin.ID())
}

// OutputIDs returns a map of output plugin ids to node ids.
func (b BasicPluginNode) OutputIDs() map[string]int64 {
	return b.outputIDs
}

// createBasicNode will create a basic node.
func createBasicNode(plugin plugin.Plugin) BasicPluginNode {
	id := createNodeID(plugin.ID())
	outputIDs := make(map[string]int64, 0)
	if plugin.CanOutput() {
		for _, output := range plugin.Outputs() {
			outputIDs[output.ID()] = createNodeID(output.ID())
		}
	}
	return BasicPluginNode{plugin, id, outputIDs}
}

// SubgraphPluginNode is a node with an embedded pipeline graph.
type SubgraphPluginNode struct {
	BasicPluginNode
}

// Subgraph will return the embedded pipeline to render in a dot graph.
func (s SubgraphPluginNode) Subgraph() graph.Graph {
	return s.plugin.(dot.Subgrapher).Subgraph()
}

// createSubgraphNode will create an embedded node.
func createSubgraphNode(plugin plugin.Plugin) SubgraphPluginNode {
	basicNode := createBasicNode(plugin)
	return SubgraphPluginNode{BasicPluginNode: basicNode}
}

// createNodeID generates a node id from a plugin id.
func createNodeID(pluginID string) int64 {
	hash := fnv.New64a()
	hash.Write([]byte(pluginID))
	return int64(hash.Sum64())
}
