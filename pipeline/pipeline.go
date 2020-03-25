package pipeline

import (
	"fmt"
	pg "github.com/bluemedora/bplogagent/plugin"
	"gonum.org/v1/gonum/graph/simple"
	"gonum.org/v1/gonum/graph/topo"
)

// Pipeline is a directed graph of connected plugins.
type Pipeline struct {
	*simple.DirectedGraph
	plugins []pg.Plugin
	built   bool
	running bool
}

// Build will build the pipeline cleanly from the supplied plugins.
func (p Pipeline) Build() error {
	if p.running {
		return fmt.Errorf("pipeline is running")
	}

	p.clearGraph()
	p.built = false

	if err := p.connectPlugins(); err != nil {
		return err
	}

	if err := p.addNodes(); err != nil {
		return err
	}

	if err := p.connectNodes(); err != nil {
		return err
	}

	p.built = true
	return nil
}

// Start will start the plugins in a pipeline in reverse topological order.
func (p Pipeline) Start() error {
	if p.running {
		return fmt.Errorf("pipeline is already running")
	}

	if !p.built {
		return fmt.Errorf("pipeline is not built")
	}

	sortedNodes, _ := topo.Sort(p)
	for i := len(sortedNodes) - 1; i >= 0; i-- {
		node, _ := sortedNodes[i].(PluginNode)
		if err := node.Plugin.Start(); err != nil {
			return fmt.Errorf("%s failed to start: %s", node.Plugin.ID(), err)
		}
	}

	p.running = true
	return nil
}

// Stop will stop the plugins in a pipeline in topological order.
func (p Pipeline) Stop() error {
	if !p.running {
		return fmt.Errorf("pipeline is not running")
	}

	sortedNodes, _ := topo.Sort(p)
	for _, node := range sortedNodes {
		pluginNode, _ := node.(PluginNode)
		pluginNode.Stop()
	}

	p.running = false
	return nil
}

// clear will clear the pipeline of connected nodes and edges.
func (p Pipeline) clearGraph() {
	p.clearEdges()
	p.clearNodes()
}

// clearEdges clears the pipeline of all edges.
func (p Pipeline) clearEdges() {
	edges := p.Edges()
	for edges.Next() {
		edge := edges.Edge()
		p.RemoveEdge(edge.From().ID(), edge.To().ID())
	}
}

// clearNodes clears the pipeline of all nodes.
func (p Pipeline) clearNodes() {
	nodes := p.Nodes()
	for nodes.Next() {
		node := nodes.Node()
		p.RemoveNode(node.ID())
	}
}

// addNodes will add the nodes to the graph.
func (p Pipeline) addNodes() error {
	for _, plugin := range p.plugins {
		node := PluginNode{plugin}
		if p.Node(node.ID()) != nil {
			return fmt.Errorf("multiple plugins with id %s", plugin.ID())
		}
		p.AddNode(node)
	}
	return nil
}

// connectNodes will connect the nodes in the graph.
func (p Pipeline) connectNodes() error {
	nodes := p.Nodes()
	for nodes.Next() {
		node, _ := nodes.Node().(PluginNode)
		if err := p.connectNode(node); err != nil {
			return fmt.Errorf("connecting %s failed: %s", node.Plugin.ID(), err)
		}
	}

	if _, err := topo.Sort(p); err != nil {
		return fmt.Errorf("pipeline has circular connections")
	}

	return nil
}

// connectNode will connect a node to its outputs in the graph.
func (p Pipeline) connectNode(node PluginNode) error {
	for pluginID, nodeID := range node.OutputIDs() {
		outputNode := p.Node(nodeID)
		if outputNode == nil {
			return fmt.Errorf("output %s is missing in pipeline", pluginID)
		}

		if p.HasEdgeFromTo(node.ID(), nodeID) {
			return fmt.Errorf("multiple connections to output %s exist", pluginID)
		}

		edge := p.NewEdge(node, outputNode)
		p.SetEdge(edge)
	}

	return nil
}

// connectPlugins will connect producers to consumers.
func (p Pipeline) connectPlugins() error {
	consumers := p.consumers()
	for _, producer := range p.producers() {
		if err := producer.SetConsumers(consumers); err != nil {
			return err
		}
	}
	return nil
}

// producers will return all producer plugins in the pipeline.
func (p Pipeline) producers() []pg.Producer {
	producers := make([]pg.Producer, 0)
	for _, plugin := range p.plugins {
		if producer, ok := plugin.(pg.Producer); ok {
			producers = append(producers, producer)
		}
	}
	return producers
}

// consumers will return all consumer plugins in the pipeline.
func (p Pipeline) consumers() []pg.Consumer {
	consumers := make([]pg.Consumer, 0)
	for _, plugin := range p.plugins {
		if consumer, ok := plugin.(pg.Consumer); ok {
			consumers = append(consumers, consumer)
		}
	}
	return consumers
}
