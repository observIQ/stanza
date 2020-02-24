package plugin

import (
	"sync"

	"github.com/bluemedora/bplogagent/entry"
	"go.uber.org/zap"
)

type SimpleProcessor interface {
	ID() string
	Input() chan entry.Entry
	Output() chan<- entry.Entry
	SetOutputs(map[string]chan<- entry.Entry) error
	ProcessEntry(entry.Entry) (entry.Entry, error)
	Logger() *zap.SugaredLogger
}

type SimpleProcessorAdapter struct {
	SimpleProcessor
}

func (s *SimpleProcessorAdapter) Start(wg *sync.WaitGroup) error {
	go func() {
		defer wg.Done()
		for {
			entry, ok := <-s.Input()
			if !ok {
				return
			}

			newEntry, err := s.ProcessEntry(entry)
			if err != nil {
				s.Logger().Warnw("Failed to process entry", "error", err)
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
