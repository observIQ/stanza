package pipeline

import (
	"hash/fnv"

	"github.com/observiq/carbon/plugin"
)

// PluginNode is a basic node that represents a plugin in a pipeline.
type PluginNode struct {
	plugin    plugin.Plugin
	id        int64
	outputIDs map[string]int64
}

// Plugin returns the plugin of the node.
func (b PluginNode) Plugin() plugin.Plugin {
	return b.plugin
}

// ID returns the node id.
func (b PluginNode) ID() int64 {
	return b.id
}

// DOTID returns the id used to represent this node in a dot graph.
func (b PluginNode) DOTID() string {
	return b.plugin.ID()
}

// OutputIDs returns a map of output plugin ids to node ids.
func (b PluginNode) OutputIDs() map[string]int64 {
	return b.outputIDs
}

// createPluginNode will create a plugin node.
func createPluginNode(plugin plugin.Plugin) PluginNode {
	id := createNodeID(plugin.ID())
	outputIDs := make(map[string]int64)
	if plugin.CanOutput() {
		for _, output := range plugin.Outputs() {
			outputIDs[output.ID()] = createNodeID(output.ID())
		}
	}
	return PluginNode{plugin, id, outputIDs}
}

// createNodeID generates a node id from a plugin id.
func createNodeID(pluginID string) int64 {
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(pluginID))
	return int64(hash.Sum64())
}
