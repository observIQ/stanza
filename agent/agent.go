package agent

import (
	"sync"

	"github.com/observiq/stanza/database"
	"github.com/observiq/stanza/pipeline"
	"go.uber.org/zap"
)

// LogAgent is an entity that handles log monitoring.
type LogAgent struct {
	database database.Database
	pipeline pipeline.Pipeline

	startOnce sync.Once
	stopOnce  sync.Once

	*zap.SugaredLogger
}

// Start will start the log monitoring process
func (a *LogAgent) Start() (err error) {
	a.startOnce.Do(func() {
		err = a.pipeline.Start()
		if err != nil {
			return
		}
	})
	return
}

// Stop will stop the log monitoring process
func (a *LogAgent) Stop() (err error) {
	a.stopOnce.Do(func() {
		err = a.pipeline.Stop()
		if err != nil {
			return
		}

		err = a.database.Close()
		if err != nil {
			return
		}
	})
	return
}
