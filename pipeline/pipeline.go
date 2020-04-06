package pipeline

import (
	"fmt"

	"github.com/bluemedora/bplogagent/plugin"
	"gonum.org/v1/gonum/graph/encoding/dot"
	"gonum.org/v1/gonum/graph/simple"
	"gonum.org/v1/gonum/graph/topo"
)

// Pipeline is a directed graph of connected plugins.
type Pipeline struct {
	Graph   *simple.DirectedGraph
	running bool
}

// Start will start the plugins in a pipeline in reverse topological order.
func (p *Pipeline) Start() error {
	if p.running {
		return nil
	}

	sortedNodes, _ := topo.Sort(p.Graph)
	for i := len(sortedNodes) - 1; i >= 0; i-- {
		plugin := sortedNodes[i].(PluginNode).Plugin()
		if err := plugin.Start(); err != nil {
			return fmt.Errorf("%s failed to start: %s", plugin.ID(), err)
		}
	}

	p.running = true
	return nil
}

// Stop will stop the plugins in a pipeline in topological order.
func (p *Pipeline) Stop() {
	if !p.running {
		return
	}

	sortedNodes, _ := topo.Sort(p.Graph)
	for _, node := range sortedNodes {
		plugin := node.(PluginNode).Plugin()
		plugin.Stop()
	}

	p.running = false
}

// MarshalDot will encode the pipeline as a dot graph.
func (p *Pipeline) MarshalDot() ([]byte, error) {
	return dot.Marshal(p.Graph, "G", "", " ")
}

// addNodes will add plugins as nodes to the supplied graph.
func addNodes(graph *simple.DirectedGraph, plugins []plugin.Plugin) error {
	for _, plugin := range plugins {
		pluginNode := createPluginNode(plugin)
		if graph.Node(pluginNode.ID()) != nil {
			return fmt.Errorf("multiple plugins with id %s", plugin.ID())
		}

		graph.AddNode(pluginNode)
	}
	return nil
}

// connectNodes will connect the nodes in the supplied graph.
func connectNodes(graph *simple.DirectedGraph) error {
	nodes := graph.Nodes()
	for nodes.Next() {
		node := nodes.Node().(PluginNode)
		if err := connectNode(graph, node); err != nil {
			return fmt.Errorf("failed to connect %s to output: %s", node.Plugin().ID(), err)
		}
	}

	// TODO: Best error message for users explaining the circular chain.
	if _, err := topo.Sort(graph); err != nil {
		return fmt.Errorf("pipeline is not acyclic: %s", err)
	}

	return nil
}

// connectNode will connect a node to its outputs in the supplied graph.
func connectNode(graph *simple.DirectedGraph, inputNode PluginNode) error {
	for outputPluginID, outputNodeID := range inputNode.OutputIDs() {
		if graph.Node(outputNodeID) == nil {
			return fmt.Errorf("output %s is missing", outputPluginID)
		}

		outputNode := graph.Node(outputNodeID).(PluginNode)
		if !outputNode.Plugin().CanProcess() {
			return fmt.Errorf("%s can not be an output", outputPluginID)
		}

		if graph.HasEdgeFromTo(inputNode.ID(), outputNodeID) {
			return fmt.Errorf("output %s already exists", outputPluginID)
		}

		edge := graph.NewEdge(inputNode, outputNode)
		graph.SetEdge(edge)
	}

	return nil
}

// setOutputs will set the outputs on plugins that can output.
func setOutputs(plugins []plugin.Plugin) error {
	for _, plugin := range plugins {
		if !plugin.CanOutput() {
			continue
		}

		if err := plugin.SetOutputs(plugins); err != nil {
			return fmt.Errorf("failed to set outputs for %s: %s", plugin.ID(), err)
		}
	}
	return nil
}

// NewPipeline creates a new pipeline of connected plugins.
func NewPipeline(plugins []plugin.Plugin) (*Pipeline, error) {
	if err := setOutputs(plugins); err != nil {
		return nil, err
	}

	graph := simple.NewDirectedGraph()
	if err := addNodes(graph, plugins); err != nil {
		return nil, err
	}

	if err := connectNodes(graph); err != nil {
		return nil, err
	}

	return &Pipeline{Graph: graph}, nil
}
