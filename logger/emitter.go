package logger

import (
	"sync"

	"go.uber.org/zap/zapcore"
)

// Receiver is a channel that receives zap logs.
type Receiver chan zapcore.Entry

// Emitter emits zap logs to a collection of receivers.
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

// Emit will emit to all receivers.
// If a receiver is full, the operation is skipped.
func (e *Emitter) Emit(entry zapcore.Entry) error {
	e.mux.RLock()
	defer e.mux.RUnlock()

	for _, receiver := range e.receivers {
		select {
		case receiver <- entry:
		default:
		}
	}

	return nil
}
