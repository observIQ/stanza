package plugin

import (
	"fmt"
	"sync"

	"github.com/bluemedora/bplogagent/entry"
	"go.uber.org/zap"
)

// Plugin is an interface that should be implemented by every plugin
type Plugin interface {

	// ID is a unique ID for a plugin instance
	ID() PluginID

	// Type is a unique ID for a plugin type
	Type() string

	// Start runs a plugin in one or more background goroutines.
	//
	// An implementation is expected to block until startup is complete,
	// and throw an error if a startup step fails. For example, if the
	// port it is configured to listen on is already in use. The wait
	// group that is passed in should be decremented when all spawned
	// goroutines have completed.
	Start(*sync.WaitGroup) error
}

// Outputter represents a plugin that outputs entries
type Outputter interface {
	Plugin
	Outputs() []Inputter
}

// Inputter represents a plugin that receives entries as input
type Inputter interface {
	Plugin
	Input() EntryChannel
}

type Stopper interface {
	Stop()
}

type PluginID string

// TODO consider whethere there is a more efficient method of copying entries
// between goroutines than a channel operation every time
type EntryChannel chan entry.Entry

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
