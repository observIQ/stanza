package plugin

import (
	"fmt"
	"hash/fnv"

	"gonum.org/v1/gonum/graph/simple"
	"gonum.org/v1/gonum/graph/topo"
)

func NewPluginConfigGraph(configs []Config) (*PluginConfigGraph, error) {
	configGraph := simple.NewDirectedGraph()
	// Build nodes
	configNodes := make([]pluginConfigNode, 0, len(configs))
	for _, config := range configs {
		configNodes = append(configNodes, pluginConfigNode{config})
	}

	// Add nodes to graph
	seenIDs := make(map[int64]struct{})
	for _, node := range configNodes {
		// Check that the node ID is unique
		if _, ok := seenIDs[node.ID()]; ok {
			return nil, fmt.Errorf("multiple configs found with id '%s'", node.PluginConfig.ID())
		} else {
			seenIDs[node.ID()] = struct{}{}
		}
		configGraph.AddNode(node)
	}

	// Connect graph
	for _, node := range configNodes {
		for outputID, outputNodeID := range node.NodeOutputIDs() {
			outputNode := configGraph.Node(outputNodeID)
			if outputNode == nil {
				return nil, fmt.Errorf("find node for output ID %s", outputID)
			}
			if outputNode.ID() == node.ID() {
				return nil, fmt.Errorf("plugin '%s' cannot output to itself", node.PluginConfig.ID())
			}
			edge := configGraph.NewEdge(node, outputNode)
			configGraph.SetEdge(edge)
		}
	}

	return &PluginConfigGraph{configGraph}, nil
}

type PluginConfigGraph struct {
	graph *simple.DirectedGraph
}

type pluginConfigNode struct {
	Config
}

func (n pluginConfigNode) NodeOutputIDs() map[PluginID]int64 {
	outputterConfig, ok := n.PluginConfig.(OutputterConfig)
	if !ok {
		return nil
	}

	ids := make(map[PluginID]int64, 0)
	for _, outputID := range outputterConfig.OutputIDs() {
		h := fnv.New64a()
		h.Write([]byte(outputID))
		ids[outputID] = int64(h.Sum64())
	}
	return ids
}

func (n pluginConfigNode) ID() int64 {
	h := fnv.New64a()
	h.Write([]byte(n.PluginConfig.ID()))
	return int64(h.Sum64())
}

func (n pluginConfigNode) DOTID() string {
	return string(n.PluginConfig.ID())
}

func (configGraph *PluginConfigGraph) Build(buildContext BuildContext) (*PluginGraph, error) {
	pluginGraph := simple.NewDirectedGraph()

	// Sort the configs topologically by outputs
	// This will fail if the graph is not acyclic
	sortedNodes, err := topo.Sort(configGraph.graph)
	if err != nil {
		// TODO make this error message more user-readable
		return nil, fmt.Errorf("order plugin dependencies: %s", err)
	}

	// Build the configs in reverse topological order
	// Plugins contains all the plugins built so far, so building
	// outputs first, and working backwards should mean all outputs
	// already exist by the time each plugin is built
	buildContext.Plugins = make(map[PluginID]Plugin)
	for i := len(sortedNodes) - 1; i >= 0; i-- { // iterate backwards
		node := sortedNodes[i]
		configNode, ok := node.(pluginConfigNode)
		if !ok {
			panic("a node was found in the graph that is not a pluginConfigNode")
		}

		plugin, err := configNode.PluginConfig.Build(buildContext)
		if err != nil {
			return nil, fmt.Errorf("build plugin with id '%s': %s", configNode.PluginConfig.ID(), err)
		}

		buildContext.Plugins[plugin.ID()] = plugin
	}

	// After building the plugins, add them to the graph
	// Check if the id is unique to ensure we don't panic
	seenIDs := make(map[int64]struct{})
	for _, plugin := range buildContext.Plugins {
		newNode := pluginNode{plugin}
		// Check that the node ID is unique
		if _, ok := seenIDs[newNode.ID()]; ok {
			return nil, fmt.Errorf("multiple plugins found with id '%s'", plugin.ID())
		} else {
			seenIDs[newNode.ID()] = struct{}{}
		}
		pluginGraph.AddNode(newNode)
	}

	// Connect the graph
	for _, plugin := range buildContext.Plugins {
		node := pluginNode{plugin}
		for outputID, outputNodeID := range node.NodeOutputIDs() {
			outputNode := pluginGraph.Node(outputNodeID)
			if outputNode == nil {
				return nil, fmt.Errorf("find node for output ID %s", outputID)
			}
			edge := pluginGraph.NewEdge(node, outputNode)
			pluginGraph.SetEdge(edge)
		}
	}

	// Warn if there is an inputter that has no outputters sending to it
	// TODO put this outside the build function
	for _, node := range sortedNodes {
		// TODO how to determine whether a config is an inputter without any special functions?
		if _, ok := node.(pluginConfigNode).PluginConfig.(InputterConfig); ok {
			outputters := configGraph.graph.To(node.ID())
			if outputters.Len() == 0 {
				buildContext.Logger.Warnw("Inputter has no outputs sending to it", "plugin_id", node.(pluginConfigNode).PluginConfig.ID())
			}
		}
	}

	return &PluginGraph{pluginGraph}, nil
}
