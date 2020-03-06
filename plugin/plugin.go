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
	Outputs() []Inputter
}

// Inputter represents a plugin that receives entries as input
type Inputter interface {
	Plugin
	Input() EntryChannel
}

// Stopper represents a plugin that should be signalled to stop
// independently on shutdown. Mostly just for sources
type Stopper interface {
	Stop()
}

type PluginID string

// TODO consider whethere there is a more efficient method of copying entries
// between goroutines than a channel operation every time
type EntryChannel chan entry.Entry
