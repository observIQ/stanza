package logger

import "go.uber.org/zap/zapcore"

// Core is a zap Core used for logging
type Core struct {
	core    zapcore.Core
	emitter *Emitter
}

// With adds contextual fields to the underlying core.
func (c *Core) With(fields []zapcore.Field) zapcore.Core {
	return &Core{
		core:    c.core.With(fields),
		emitter: c.emitter,
	}
}

// Enabled will check if the supplied log level is enabled.
func (c *Core) Enabled(level zapcore.Level) bool {
	return c.core.Enabled(level)
}

// Check checks the entry and determines if the core should write it.
func (c *Core) Check(zapEntry zapcore.Entry, checkedEntry *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if !c.Enabled(zapEntry.Level) {
		return checkedEntry
	}
	return checkedEntry.AddCore(zapEntry, c)
}

// Write sends an entry to the emitter before logging.
func (c *Core) Write(zapEntry zapcore.Entry, fields []zapcore.Field) error {
	stanzaEntry := parseEntry(zapEntry, fields)
	c.emitter.emit(stanzaEntry)
	return c.core.Write(zapEntry, fields)
}

// Sync will sync the underlying core.
func (c *Core) Sync() error {
	return c.core.Sync()
}

// newCore creates a new core.
func newCore(core zapcore.Core, emitter *Emitter) *Core {
	return &Core{
		core:    core,
		emitter: emitter,
	}
}
