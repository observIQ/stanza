package plugin

import (
	"sync"

	"github.com/bluemedora/bplogagent/entry"
)

type Plugin interface {
	ID() string
	Type() string
	Start(*sync.WaitGroup) error
}

type Source interface {
	Plugin
	Outputter
	Stop()
}

// TODO do we even need this interface? Might be better to just have source, outputter, and inputter
// with inputter embedded
type Processor interface {
	Plugin
	Outputter
	Inputter
}

// TODO do we even need this interface?
type Destination interface {
	Plugin
	Inputter
}

type Outputter interface {
	SetOutputs(map[string]chan<- entry.Entry) error
	Outputs() []chan<- entry.Entry
}

type Inputter interface {
	Input() chan entry.Entry
}
