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

type Outputter interface {
	Plugin
	SetOutputs(map[string]chan<- entry.Entry) error
	Outputs() []chan<- entry.Entry
}

type Inputter interface {
	Plugin
	Input() chan entry.Entry
}

type Source interface {
	Outputter
	Stop()
}
