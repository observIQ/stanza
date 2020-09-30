package logger

import (
	"go.uber.org/zap"
)

// Logger is a wrapped logger used by the stanza agent.
type Logger struct {
	*zap.SugaredLogger
	*Emitter
}

// New will create a new logger.
func New(sugared *zap.SugaredLogger) *Logger {
	baseLogger := sugared.Desugar()
	emitter := newEmitter()
	core := newCore(baseLogger.Core(), emitter)
	wrappedLogger := zap.New(core).Sugar()

	return &Logger{
		wrappedLogger,
		emitter,
	}
}
