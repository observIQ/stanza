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
	pipeline pipeline.Pipeline
	running  bool
}

// Start will start the log monitoring process.
func (a *LogAgent) Start() error {
	if a.running {
		return nil
	}

	database, err := a.openDatabase()
	if err != nil {
		return fmt.Errorf("open database: %s", err)
	}
	a.database = database

	plugins, err := a.buildPlugins()
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

	if dotGraph, err := pipeline.MarshalDot(); err != nil {
		a.Infof("Pipeline:\n%s", dotGraph)
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
	a.closeDatabase()
	a.running = false
	a.Info("Agent stopped")
}

// Status will return the status of the agent.
func (a *LogAgent) Status() struct{} {
	return struct{}{}
}

// buildPlugins builds the plugins listed in the agent config.
func (a *LogAgent) buildPlugins() ([]pg.Plugin, error) {
	context := a.buildContext()
	plugins := make([]pg.Plugin, 0)

	for _, pluginConfig := range a.Config.Plugins {
		plugin, err := pluginConfig.Build(context)
		if err != nil {
			return plugins, fmt.Errorf("failed to build %s: %s", pluginConfig.ID(), err)
		}
		plugins = append(plugins, plugin)
	}

	return plugins, nil
}

// buildContext will create a build context for building plugins.
func (a *LogAgent) buildContext() pg.BuildContext {
	return pg.BuildContext{
		Logger:   a.SugaredLogger,
		Bundles:  bundle.GetBundleDefinitions(a.Config.BundlePath, a.SugaredLogger),
		Database: a.database,
	}
}

// openDatabase will open a connection to the database.
func (a *LogAgent) openDatabase() (*bbolt.DB, error) {
	file := a.databaseFile()
	options := &bbolt.Options{Timeout: 1 * time.Second}
	return bbolt.Open(file, 0666, options)
}

// closeDatabase will close the database connection.
func (a *LogAgent) closeDatabase() {
	if a.database != nil {
		if err := a.database.Close(); err != nil {
			a.Errorf("Failed to close database: %s", err)
		}
	}
}

// databaseFile returns the location of the database.
func (a *LogAgent) databaseFile() string {
	if a.Config.DatabaseFile != "" {
		return a.Config.DatabaseFile
	}

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
