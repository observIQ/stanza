package plugin

import (
	"sync"

	"github.com/bluemedora/bplogagent/entry"
)

type SimpleProcessor interface {
	ID() string
	Input() chan entry.Entry
	Output() chan<- entry.Entry
	SetOutputs(map[string]chan<- entry.Entry) error
	ProcessEntry(entry.Entry) (entry.Entry, error)
}

type SimpleProcessorAdapter struct {
	SimpleProcessor
}

func (s *SimpleProcessorAdapter) Start(wg *sync.WaitGroup) error {
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			entry, ok := <-s.Input()
			if !ok {
				return
			}

			newEntry, err := s.ProcessEntry(entry)
			if err != nil {
				// TODO handle processing errors
				continue
			}

			s.Output() <- newEntry
		}
	}()

	return nil
}

func (s *SimpleProcessorAdapter) Outputs() []chan<- entry.Entry {
	return []chan<- entry.Entry{s.Output()}
}
