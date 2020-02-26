package plugin

import (
	"sync"

	"github.com/bluemedora/bplogagent/entry"
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
	// TODO these should probably take arrays of inputters rather than maps
	// Maybe a specific type like PluginRegistry that has a FindByID method
	SetOutputs(map[PluginID]EntryChannel) error
	Outputs() map[PluginID]EntryChannel
}

// Inputter represents a plugin that receives entries
type Inputter interface {
	Plugin
	Input() EntryChannel
}

type Source interface {
	Outputter
	Stop()
}

type PluginID string
type EntryChannel chan entry.Entry
