package plugin

import (
	"fmt"
	"hash/fnv"
	"sync"

	"github.com/bluemedora/bplogagent/bundle"
	"github.com/bluemedora/bplogagent/entry"
	"go.uber.org/zap"
	"gonum.org/v1/gonum/graph/encoding/dot"
	"gonum.org/v1/gonum/graph/simple"
	"gonum.org/v1/gonum/graph/topo"
)

type BuildContext struct {
	Plugins      map[PluginID]Plugin
	Bundles      []*bundle.BundleDefinition
	BundleInput  EntryChannel
	BundleOutput EntryChannel
	Logger       *zap.SugaredLogger
}

type pluginConfigNode struct {
	config PluginConfig
}

func (n pluginConfigNode) OutputIDs() map[PluginID]int64 {
	outputterConfig, ok := n.config.(OutputterConfig)
	if !ok {
		return nil
	}

	ids := make(map[PluginID]int64, 0)
	for _, outputID := range outputterConfig.Outputs() {
		h := fnv.New64a()
		h.Write([]byte(outputID))
		ids[outputID] = int64(h.Sum64())
	}
	return ids
}

func (n pluginConfigNode) ID() int64 {
	h := fnv.New64a()
	h.Write([]byte(n.config.ID()))
	return int64(h.Sum64())
}

func (n pluginConfigNode) DOTID() string {
	return string(n.config.ID())
}

func BuildPlugins(configs []PluginConfig, buildContext BuildContext) ([]Plugin, error) {
	// Construct the graph from the configs
	configGraph := simple.NewDirectedGraph()
	err := addConfigsToGraph(configGraph, configs)
	if err != nil {
		return nil, fmt.Errorf("failed to build config graph: %s", err)
	}

	marshalled, err := dot.Marshal(configGraph, "G", "", " ")
	if err != nil {
		buildContext.Logger.Info("Failed to marshal the config graph: %s", err)
	}
	buildContext.Logger.Info("Created a graph:\n", string(marshalled))

	// Sort the configs topologically by outputs
	// This will fail if the graph is not acyclic
	sortedNodes, err := topo.Sort(configGraph)
	if err != nil {
		// TODO make this error message more user-readable
		return nil, fmt.Errorf("failed to order plugin dependencies: %s", err)
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

		plugin, err := configNode.config.Build(buildContext)
		if err != nil {
			return nil, fmt.Errorf("failed to build plugin with id '%s': %s", configNode.config.ID(), err)
		}

		buildContext.Plugins[plugin.ID()] = plugin
	}

	// Warn if there is an inputter that has no outputters sending to it
	for _, node := range sortedNodes {
		if _, ok := node.(pluginConfigNode).config.(InputterConfig); ok {
			outputters := configGraph.To(node.ID())
			if outputters.Len() == 0 {
				buildContext.Logger.Warnw("Inputter has no outputs sending to it", "plugin_id", node.(pluginConfigNode).config.ID())
			}
		}
	}

	// Convert from a map to a slice
	pluginSlice := make([]Plugin, 0, len(buildContext.Plugins))
	for _, plugin := range buildContext.Plugins {
		pluginSlice = append(pluginSlice, plugin)
	}

	return pluginSlice, nil
}

func addConfigsToGraph(configGraph *simple.DirectedGraph, configs []PluginConfig) error {
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
			return fmt.Errorf("multiple configs found with id '%s'", node.config.ID())
		} else {
			seenIDs[node.ID()] = struct{}{}
		}
		configGraph.AddNode(node)
	}

	// Connect graph
	for _, node := range configNodes {
		for outputID, outputNodeID := range node.OutputIDs() {
			outputNode := configGraph.Node(outputNodeID)
			if outputNode == nil {
				return fmt.Errorf("failed to find node for output ID %s", outputID)
			}
			edge := configGraph.NewEdge(node, outputNode)
			configGraph.SetEdge(edge)
		}
	}

	return nil
}

// TODO put this in its own file?
func StartPlugins(plugins []Plugin, pluginWg *sync.WaitGroup, logger *zap.SugaredLogger) error {
	closer := &inputChannelCloser{
		waitGroupMap:  make(map[chan<- entry.Entry]*sync.WaitGroup),
		SugaredLogger: logger,
	}
	defer closer.StartChannelClosers()

	for _, plugin := range plugins {
		if inputter, ok := plugin.(Inputter); ok {
			closer.AddInputter(inputter)
		}
	}

	for _, plugin := range plugins {
		// Start the plugin
		wg := new(sync.WaitGroup)
		wg.Add(1)
		logger.Debugw("Starting plugin", "plugin_id", plugin.ID(), "plugin_type", plugin.Type())
		err := plugin.Start(wg)
		if err != nil {
			return fmt.Errorf("failed to start plugin with ID '%s': %s", plugin.ID(), err)
		}

		// Register a handler for the global plugin waitgroup
		pluginWg.Add(1)
		go func(plugin Plugin, wg *sync.WaitGroup) {
			wg.Wait()
			logger.Debugw("Plugin stopped", "id", plugin.ID())
			pluginWg.Done()
		}(plugin, wg)

		// If it's an outputter, close its output channels
		if outputter, ok := plugin.(Outputter); ok {
			closer.AddOutputter(outputter)
			go func(wg *sync.WaitGroup, outputter Outputter) {
				wg.Wait()
				closer.Done(outputter)
			}(wg, outputter)
		}
	}

	return nil
}

type inputChannelCloser struct {
	waitGroupMap map[chan<- entry.Entry]*sync.WaitGroup
	sync.Mutex
	*zap.SugaredLogger
}

func (i *inputChannelCloser) AddInputter(inputter Inputter) {
	i.Lock()
	_, ok := i.waitGroupMap[inputter.Input()]
	if ok {
		panic("waitgroup already created for inputter")
	} else {
		newWg := new(sync.WaitGroup)
		i.waitGroupMap[inputter.Input()] = newWg
	}
	i.Unlock()
}

func (i *inputChannelCloser) AddOutputter(outputter Outputter) {
	i.Lock()
	for _, inputter := range outputter.Outputs() {
		wg, ok := i.waitGroupMap[inputter.Input()]
		if ok {
			wg.Add(1)
		} else {
			panic("no waitgroup found for inputter")
		}
	}
	i.Unlock()
}

func (i *inputChannelCloser) Done(outputter Outputter) {
	i.Lock()
	for _, inputter := range outputter.Outputs() {
		wg, ok := i.waitGroupMap[inputter.Input()]
		if ok {
			wg.Done()
		} else {
			panic("called Done() for a channel that doesn't exist")
		}
	}
	i.Unlock()
}

func (i *inputChannelCloser) StartChannelClosers() {
	i.Lock()
	for channel, waitGroup := range i.waitGroupMap {
		go func(channel chan<- entry.Entry, wg *sync.WaitGroup) {
			wg.Wait()
			close(channel)
		}(channel, waitGroup)
	}
	i.Unlock()
}
