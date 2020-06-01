package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/bluemedora/bplogagent/errors"
	"github.com/bluemedora/bplogagent/pipeline"
	pg "github.com/bluemedora/bplogagent/plugin"
	_ "github.com/bluemedora/bplogagent/plugin/builtin" // register plugins
	"go.etcd.io/bbolt"
	"go.uber.org/zap"
)

// LogAgent is an entity that handles log monitoring.
type LogAgent struct {
	Config *Config
	*zap.SugaredLogger

	database *bbolt.DB
	pipeline *pipeline.Pipeline
	running  bool
}

// Start will start the log monitoring process.
func (a *LogAgent) Start() error {
	if a.running {
		return nil
	}

	database, err := openDatabase(a.Config.DatabaseFile)
	if err != nil {
		a.Errorw("Failed to open database", zap.Any("error", err))
		return err
	}
	a.database = database

	buildContext := newBuildContext(a.SugaredLogger, database)
	pipeline, err := a.Config.Pipeline.BuildPipeline(buildContext)
	if err != nil {
		return errors.Wrap(err, "build pipeline")
	}
	a.pipeline = pipeline

	err = a.pipeline.Start()
	if err != nil {
		a.Errorw("Failed to start pipeline", zap.Any("error", err))
		return err
	}

	a.running = true
	a.Info("Agent started")
	return nil
}

// Stop will stop the log monitoring process.
func (a *LogAgent) Stop() {
	if !a.running {
		return
	}

	a.pipeline.Stop()
	a.pipeline = nil

	a.database.Close()
	a.database = nil

	a.running = false
	a.Info("Agent stopped")
}

// Status will return the status of the agent.
func (a *LogAgent) Status() struct{} {
	return struct{}{}
}

// newBuildContext will create a new build context for building plugins.
func newBuildContext(logger *zap.SugaredLogger, database *bbolt.DB) pg.BuildContext {
	return pg.BuildContext{
		Logger:   logger,
		Database: database,
	}
}

// openDatabase will open and create a database.
func openDatabase(file string) (*bbolt.DB, error) {
	if file == "" {
		file = defaultDatabaseFile()
	}

	if _, err := os.Stat(filepath.Dir(file)); err != nil {
		if os.IsNotExist(err) {
			err := os.MkdirAll(filepath.Dir(file), 0666)
			if err != nil {
				return nil, fmt.Errorf("creating database directory: %s", err)
			}
		} else {
			return nil, err
		}
	}

	options := &bbolt.Options{Timeout: 1 * time.Second}
	return bbolt.Open(file, 0666, options)
}

// defaultDatabaseFile returns the default location of the database.
func defaultDatabaseFile() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		return filepath.Join(".", "bplogagent.db")
	}
	return filepath.Join(dir, "bplogagent.db")
}

// NewLogAgent creates a new log agent.
func NewLogAgent(cfg *Config, logger *zap.SugaredLogger) *LogAgent {
	return &LogAgent{
		Config:        cfg,
		SugaredLogger: logger,
	}
}
