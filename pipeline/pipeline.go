package pipeline

import (
	"fmt"
	"strings"

	"github.com/observiq/carbon/errors"
	"github.com/observiq/carbon/plugin"
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
		plugin := sortedNodes[i].(OperatorNode).Operator()
		plugin.Logger().Debug("Starting plugin")
		if err := plugin.Start(); err != nil {
			return err
		}
		plugin.Logger().Debug("Started plugin")
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
		plugin := node.(OperatorNode).Operator()
		plugin.Logger().Debug("Stopping plugin")
		_ = plugin.Stop()
		plugin.Logger().Debug("Stopped plugin")
	}

	p.running = false
}

// MarshalDot will encode the pipeline as a dot graph.
func (p *Pipeline) MarshalDot() ([]byte, error) {
	return dot.Marshal(p.Graph, "G", "", " ")
}

// addNodes will add plugins as nodes to the supplied graph.
func addNodes(graph *simple.DirectedGraph, plugins []plugin.Operator) error {
	for _, plugin := range plugins {
		pluginNode := createOperatorNode(plugin)
		if graph.Node(pluginNode.ID()) != nil {
			return errors.NewError(
				fmt.Sprintf("plugin with id '%s' already exists in pipeline", pluginNode.Operator().ID()),
				"ensure that each plugin has a unique `type` or `id`",
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
		node := nodes.Node().(OperatorNode)
		if err := connectNode(graph, node); err != nil {
			return err
		}
	}

	if _, err := topo.Sort(graph); err != nil {
		return errors.NewError(
			"pipeline has a circular dependency",
			"ensure that all plugins are connected in a straight, acyclic line",
			"cycles", unorderableToCycles(err.(topo.Unorderable)),
		)
	}

	return nil
}

// connectNode will connect a node to its outputs in the supplied graph.
func connectNode(graph *simple.DirectedGraph, inputNode OperatorNode) error {
	for outputOperatorID, outputNodeID := range inputNode.OutputIDs() {
		if graph.Node(outputNodeID) == nil {
			return errors.NewError(
				"plugins cannot be connected, because the output does not exist in the pipeline",
				"ensure that the output plugin is defined",
				"input_plugin", inputNode.Operator().ID(),
				"output_plugin", outputOperatorID,
			)
		}

		outputNode := graph.Node(outputNodeID).(OperatorNode)
		if !outputNode.Operator().CanProcess() {
			return errors.NewError(
				"plugins cannot be connected, because the output plugin can not process logs",
				"ensure that the output plugin can process logs (like a parser or destination)",
				"input_plugin", inputNode.Operator().ID(),
				"output_plugin", outputOperatorID,
			)
		}

		if graph.HasEdgeFromTo(inputNode.ID(), outputNodeID) {
			return errors.NewError(
				"plugins cannot be connected, because a connection already exists",
				"ensure that only a single connection exists between the two plugins",
				"input_plugin", inputNode.Operator().ID(),
				"output_plugin", outputOperatorID,
			)
		}

		edge := graph.NewEdge(inputNode, outputNode)
		graph.SetEdge(edge)
	}

	return nil
}

// setOperatorOutputs will set the outputs on plugins that can output.
func setOperatorOutputs(plugins []plugin.Operator) error {
	for _, plugin := range plugins {
		if !plugin.CanOutput() {
			continue
		}

		if err := plugin.SetOutputs(plugins); err != nil {
			return errors.WithDetails(err, "plugin_id", plugin.ID())
		}
	}
	return nil
}

// NewPipeline creates a new pipeline of connected plugins.
func NewPipeline(plugins []plugin.Operator) (*Pipeline, error) {
	if err := setOperatorOutputs(plugins); err != nil {
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

func unorderableToCycles(err topo.Unorderable) string {
	var cycles strings.Builder
	for i, cycle := range err {
		if i != 0 {
			cycles.WriteByte(',')
		}
		cycles.WriteByte('(')
		for _, node := range cycle {
			cycles.WriteString(node.(OperatorNode).plugin.ID())
			cycles.Write([]byte(` -> `))
		}
		cycles.WriteString(cycle[0].(OperatorNode).plugin.ID())
		cycles.WriteByte(')')
	}
	return cycles.String()
}
