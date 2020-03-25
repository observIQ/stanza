package pipeline

import (
	"hash/fnv"

	"github.com/bluemedora/bplogagent/plugin"
)

// PluginNode is a node that represents a plugin in a pipeline.
type PluginNode struct {
	plugin.Plugin
}

// ID returns the node id.
func (n PluginNode) ID() int64 {
	return createNodeID(n.Plugin.ID())
}

// DOTID returns the id used to represent this node in a dot graph.
func (n PluginNode) DOTID() string {
	return string(n.Plugin.ID())
}

// OutputIDs returns a map of connected ids
// in the form of plugin id to node id.
func (n PluginNode) OutputIDs() map[string]int64 {
	ids := make(map[string]int64, 0)
	if producer, ok := n.Plugin.(plugin.Producer); ok {
		for _, consumer := range producer.Consumers() {
			ids[consumer.ID()] = createNodeID(consumer.ID())
		}
	}
	return ids
}

// createNodeID generates a node id from a plugin id.
func createNodeID(pluginID string) int64 {
	hash := fnv.New64a()
	hash.Write([]byte(pluginID))
	return int64(hash.Sum64())
}
