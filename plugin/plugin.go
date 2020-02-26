package plugin

import (
	"sync"

	"github.com/bluemedora/bplogagent/entry"
)

type Plugin interface {
	ID() PluginID
	Type() string
	Start(*sync.WaitGroup) error
}

type Outputter interface {
	Plugin
	// TODO these should probably take arrays of inputters rather than maps
	// Maybe a specific type like PluginRegistry that has a FindByID method
	SetOutputs(map[PluginID]EntryChannel) error
	Outputs() map[PluginID]EntryChannel
}

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
