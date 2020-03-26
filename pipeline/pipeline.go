package pipeline

import (
	"fmt"

	pg "github.com/bluemedora/bplogagent/plugin"
	"gonum.org/v1/gonum/graph/encoding/dot"
	"gonum.org/v1/gonum/graph/simple"
	"gonum.org/v1/gonum/graph/topo"
)

// Pipeline is a directed graph of connected plugins.
type Pipeline struct {
	graph   *simple.DirectedGraph
	running bool
}

// Start will start the plugins in a pipeline in reverse topological order.
func (p *Pipeline) Start() error {
	if p.running {
		return nil
	}

	sortedNodes, _ := topo.Sort(p.graph)
	for i := len(sortedNodes) - 1; i >= 0; i-- {
		node := sortedNodes[i].(PluginNode)
		if err := node.Plugin.Start(); err != nil {
			return fmt.Errorf("%s failed to start: %s", node.Plugin.ID(), err)
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

	sortedNodes, _ := topo.Sort(p.graph)
	for _, node := range sortedNodes {
		pluginNode := node.(PluginNode)
		pluginNode.Stop()
	}

	p.running = false
}

// MarshalDot will encode the pipeline as a dot graph.
func (p *Pipeline) MarshalDot() ([]byte, error) {
	return dot.Marshal(p.graph, "G", "", " ")
}

// addNodes will add plugins as nodes to the supplied graph.
func addNodes(graph *simple.DirectedGraph, plugins []pg.Plugin) error {
	for _, plugin := range plugins {
		node := PluginNode{plugin}
		if graph.Node(node.ID()) != nil {
			return fmt.Errorf("multiple plugins with id %s", plugin.ID())
		}
		graph.AddNode(node)
	}
	return nil
}

// connectNodes will connect the nodes in the supplied graph.
func connectNodes(graph *simple.DirectedGraph) error {
	nodes := graph.Nodes()
	for nodes.Next() {
		node := nodes.Node().(PluginNode)
		if err := connectNode(graph, node); err != nil {
			return fmt.Errorf("connecting %s failed: %s", node.Plugin.ID(), err)
		}
	}

	if _, err := topo.Sort(graph); err != nil {
		return fmt.Errorf("graph is not acyclic: %s", err)
	}

	return nil
}

// connectNode will connect a node to its outputs in the supplied graph.
func connectNode(graph *simple.DirectedGraph, node PluginNode) error {
	for pluginID, nodeID := range node.OutputIDs() {
		outputNode := graph.Node(nodeID)
		if outputNode == nil {
			return fmt.Errorf("output %s is missing", pluginID)
		}

		if graph.HasEdgeFromTo(node.ID(), nodeID) {
			return fmt.Errorf("multiple connections to %s exist", pluginID)
		}

		edge := graph.NewEdge(node, outputNode)
		graph.SetEdge(edge)
	}

	return nil
}

// connectPlugins will connect producers to consumers.
func connectPlugins(plugins []pg.Plugin) error {
	consumers := consumers(plugins)
	for _, producer := range producers(plugins) {
		if err := producer.SetConsumers(consumers); err != nil {
			return err
		}
	}
	return nil
}

// producers will return only producers from a list.
func producers(plugins []pg.Plugin) []pg.Producer {
	producers := make([]pg.Producer, 0)
	for _, plugin := range plugins {
		if producer, ok := plugin.(pg.Producer); ok {
			producers = append(producers, producer)
		}
	}
	return producers
}

// consumers will return only consumers from a list.
func consumers(plugins []pg.Plugin) []pg.Consumer {
	consumers := make([]pg.Consumer, 0)
	for _, plugin := range plugins {
		if consumer, ok := plugin.(pg.Consumer); ok {
			consumers = append(consumers, consumer)
		}
	}
	return consumers
}

// NewPipeline creates a new pipeline of connected plugins.
func NewPipeline(plugins []pg.Plugin) (*Pipeline, error) {
	if err := connectPlugins(plugins); err != nil {
		return nil, err
	}

	graph := simple.NewDirectedGraph()
	if err := addNodes(graph, plugins); err != nil {
		return nil, err
	}

	if err := connectNodes(graph); err != nil {
		return nil, err
	}

	return &Pipeline{graph: graph}, nil
}
