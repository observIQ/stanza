package plugin

import (
	"fmt"
	"hash/fnv"

	"gonum.org/v1/gonum/graph/encoding/dot"
	"gonum.org/v1/gonum/graph/simple"
	"gonum.org/v1/gonum/graph/topo"
)

type PluginGraph struct {
	Graph *simple.DirectedGraph
}

type pluginNode struct {
	Plugin
}

func (n pluginNode) NodeOutputIDs() map[PluginID]int64 {
	outputterConfig, ok := n.Plugin.(Outputter)
	if !ok {
		return nil
	}

	ids := make(map[PluginID]int64, 0)
	for _, output := range outputterConfig.Outputs() {
		h := fnv.New64a()
		h.Write([]byte(output.ID()))
		ids[output.ID()] = int64(h.Sum64())
	}
	return ids
}

func (n pluginNode) ID() int64 {
	h := fnv.New64a()
	h.Write([]byte(n.Plugin.ID()))
	return int64(h.Sum64())
}

func (n pluginNode) DOTID() string {
	return string(n.Plugin.ID())
}

func (pluginGraph *PluginGraph) Start() error {
	sortedPlugins := pluginGraph.SortedPlugins()

	// Start plugins, destinations first
	for i := len(sortedPlugins) - 1; i >= 0; i-- {
		if starter, ok := sortedPlugins[i].(Starter); ok {
			err := starter.Start()
			if err != nil {
				return fmt.Errorf("start plugin '%s': %s", starter.ID(), err)
			}
		}
	}

	return nil
}

func (pluginGraph *PluginGraph) Stop() {
	sortedPlugins := pluginGraph.SortedPlugins()

	// Stop plugins, sources first
	for _, plugin := range sortedPlugins {
		if stopper, ok := plugin.(Stopper); ok {
			stopper.Stop()
		}
	}
}

// Sorted returns a topographically sorted list of plugins from
// sources to outputs
// TODO this should never error, since we sorted the configs
// during build time, but I should think about this more closely
func (pluginGraph *PluginGraph) SortedPlugins() []Plugin {
	sortedNodes, err := topo.Sort(pluginGraph.Graph)
	if err != nil {
		panic(err)
	}

	plugins := make([]Plugin, len(sortedNodes))
	for i, node := range sortedNodes {
		plugins[i] = node.(pluginNode).Plugin
	}

	return plugins

}

func (pluginGraph *PluginGraph) MarshalDot() ([]byte, error) {
	return dot.Marshal(pluginGraph.Graph, "G", "", " ")
}
