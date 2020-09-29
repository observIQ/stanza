package logger

import (
	"go.uber.org/zap"
)

// Logger is a wrapped logger used by the stanza agent.
type Logger struct {
	*zap.SugaredLogger
	emitter *Emitter
}

// AddReceiver will add a receiver to the logger.
func (l *Logger) AddReceiver(receiver Receiver) {
	l.emitter.AddReceiver(receiver)
}

// New will create a new logger.
func New(sugared *zap.SugaredLogger) *Logger {
	emitter := &Emitter{}
	hooks := zap.Hooks(emitter.Emit)
	base := sugared.Desugar().WithOptions(hooks)

	return &Logger{
		base.Sugar(),
		emitter,
	}
}
