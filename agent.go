package bplogagent

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/bluemedora/bplogagent/bundle"
	"github.com/bluemedora/bplogagent/config"
	"github.com/bluemedora/bplogagent/pipeline"
	pg "github.com/bluemedora/bplogagent/plugin"
	_ "github.com/bluemedora/bplogagent/plugin/builtin" // register plugins
	"go.etcd.io/bbolt"
	"go.uber.org/zap"
)

// LogAgent is an entity that handles log monitoring.
type LogAgent struct {
	Config config.Config
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
		return fmt.Errorf("open database: %s", err)
	}
	a.database = database

	buildContext := newBuildContext(a.SugaredLogger, database, a.Config.BundlePath)
	plugins, err := pg.BuildPlugins(a.Config.Plugins, buildContext)
	if err != nil {
		return fmt.Errorf("build plugins: %s", err)
	}

	pipeline, err := pipeline.NewPipeline(plugins)
	if err != nil {
		return fmt.Errorf("build pipeline: %s", err)
	}
	a.pipeline = pipeline

	err = a.pipeline.Start()
	if err != nil {
		return fmt.Errorf("start pipeline: %s", err)
	}

	if dotGraph, err := pipeline.MarshalDot(); err == nil {
		a.Infof("Pipeline:\n%s", dotGraph)
	} else {
		a.Errorf("Failed to render dot: %s", err)
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
func newBuildContext(logger *zap.SugaredLogger, database *bbolt.DB, bundlePath string) pg.BuildContext {
	return pg.BuildContext{
		Logger:   logger,
		Bundles:  bundle.GetBundleDefinitions(bundlePath, logger),
		Database: database,
	}
}

// openDatabase will open and create a database.
func openDatabase(file string) (*bbolt.DB, error) {
	if file == "" {
		file = defaultDatabaseFile()
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
func NewLogAgent(cfg config.Config, logger *zap.SugaredLogger) *LogAgent {
	return &LogAgent{
		Config:        cfg,
		SugaredLogger: logger,
	}
}
