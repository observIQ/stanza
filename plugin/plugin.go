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

type Processor interface {
	Plugin
	Outputter
	Inputter
}

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
