package plugin

import (
	bpla "github.com/bluemedora/bplogagent"
)

type Source interface {
	Shutdown()
}

type EntryProcessor interface {
	ProcessEntry(bpla.Entry) ([]EntryProcessStep, error)
}

type EntryProcessStep struct {
	Entry     bpla.Entry
	Processor EntryProcessor
}
