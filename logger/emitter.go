package logger

import (
	"sync"

	"github.com/observiq/stanza/entry"
)

// Receiver is a channel that receives internal stanza logs.
type Receiver chan entry.Entry

// Emitter emits internal logs to registered receivers.
type Emitter struct {
	receivers []Receiver
	mux       sync.RWMutex
}

// AddReceiver will add a receiver to the emitter.
func (e *Emitter) AddReceiver(receiver Receiver) {
	e.mux.Lock()
	e.receivers = append(e.receivers, receiver)
	e.mux.Unlock()
}

// Emit emits an entry to all receivers.
func (e *Emitter) emit(entry entry.Entry) {
	e.mux.RLock()
	defer e.mux.RUnlock()
	for _, receiver := range e.receivers {
		select {
		case receiver <- entry:
		default:
		}
	}
}

// newEmitter creates a new emitter.
func newEmitter() *Emitter {
	return &Emitter{
		receivers: make([]Receiver, 0, 2),
	}
}
