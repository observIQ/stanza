package pipeline

import (
	"github.com/bluemedora/bplogagent/errors"
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
			return err
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
			return errors.NewError(
				"Plugin already exists in the pipeline.",
				"Ensure that all plugins are defined only once with a unique id.",
				"plugin_id", pluginNode.Plugin().ID(),
			)
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
			return err
		}
	}

	// TODO: Best error message for users explaining the circular chain.
	if _, err := topo.Sort(graph); err != nil {
		return errors.NewError(
			"Pipeline has a circular dependency.",
			"Ensure that all plugins are connected in a straight, acyclic line.",
			"raw_error", err.Error(),
		)
	}

	return nil
}

// connectNode will connect a node to its outputs in the supplied graph.
func connectNode(graph *simple.DirectedGraph, inputNode PluginNode) error {
	for outputPluginID, outputNodeID := range inputNode.OutputIDs() {
		if graph.Node(outputNodeID) == nil {
			return errors.NewError(
				"Plugins cannot be connected, because the output does not exist in the pipeline.",
				"Ensure that the output plugin is defined.",
				"input_plugin", inputNode.Plugin().ID(),
				"output_plugin", outputPluginID,
			)
		}

		outputNode := graph.Node(outputNodeID).(PluginNode)
		if !outputNode.Plugin().CanProcess() {
			return errors.NewError(
				"Plugins cannot be connected, because the output plugin can not process logs.",
				"Ensure that the output plugin can process logs (like a parser or destination).",
				"input_plugin", inputNode.Plugin().ID(),
				"output_plugin", outputPluginID,
			)
		}

		if graph.HasEdgeFromTo(inputNode.ID(), outputNodeID) {
			return errors.NewError(
				"Plugins cannot be connected, because a connection already exists.",
				"Ensure that only a single connection exists between the two plugins",
				"input_plugin", inputNode.Plugin().ID(),
				"output_plugin", outputPluginID,
			)
		}

		edge := graph.NewEdge(inputNode, outputNode)
		graph.SetEdge(edge)
	}

	return nil
}

// setPluginOutputs will set the outputs on plugins that can output.
func setPluginOutputs(plugins []plugin.Plugin) error {
	for _, plugin := range plugins {
		if !plugin.CanOutput() {
			continue
		}

		if err := plugin.SetOutputs(plugins); err != nil {
			return err
		}
	}
	return nil
}

// NewPipeline creates a new pipeline of connected plugins.
func NewPipeline(plugins []plugin.Plugin) (*Pipeline, error) {
	if err := setPluginOutputs(plugins); err != nil {
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
