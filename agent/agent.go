package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/observiq/carbon/errors"
	"github.com/observiq/carbon/operator"
	_ "github.com/observiq/carbon/operator/builtin" // register operators
	"github.com/observiq/carbon/pipeline"
	"go.etcd.io/bbolt"
	"go.uber.org/zap"
)

// LogAgent is an entity that handles log monitoring.
type LogAgent struct {
	Config    *Config
	PluginDir string
	Database  string
	*zap.SugaredLogger

	buildParams map[string]interface{}
	database    operator.Database
	pipeline    *pipeline.Pipeline
	running     bool

	startOnce sync.Once
	stopOnce  sync.Once
}

// Start will start the log monitoring process.
func (a *LogAgent) Start() error {
	var err error
	a.startOnce.Do(func() {
		database, err := OpenDatabase(a.Database)
		if err != nil {
			err = errors.Wrap(err, "open database")
			return
		}
		a.database = database

		registry, err := operator.NewPluginRegistry(a.PluginDir)
		if err != nil {
			a.Errorw("Failed to load plugin registry", zap.Any("error", err))
		}

		buildContext := operator.BuildContext{
			PluginRegistry: registry,
			Logger:         a.SugaredLogger,
			Database:       a.database,
			Parameters:     a.buildParams,
		}

		pipeline, err := a.Config.Pipeline.BuildPipeline(buildContext)
		if err != nil {
			err = errors.Wrap(err, "build pipeline")
			return
		}
		a.pipeline = pipeline

		err = a.pipeline.Start()
		if err != nil {
			err = errors.Wrap(err, "start pipeline")
			return
		}

		a.Info("Agent started")
	})

	return err
}

// Stop will stop the log monitoring process.
func (a *LogAgent) Stop() {
	a.stopOnce.Do(func() {
		a.pipeline.Stop()
		a.database.Close()
	})

	a.Info("Agent stopped")
}

// OpenDatabase will open and create a database.
func OpenDatabase(file string) (operator.Database, error) {
	if file == "" {
		return operator.NewStubDatabase(), nil
	}

	if _, err := os.Stat(filepath.Dir(file)); err != nil {
		if os.IsNotExist(err) {
			err := os.MkdirAll(filepath.Dir(file), 0755)
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

// NewLogAgent creates a new carbon log agent.
func NewLogAgent(cfg *Config, logger *zap.SugaredLogger, pluginDir, databaseFile string) *LogAgent {
	return &LogAgent{
		Config:        cfg,
		SugaredLogger: logger,
		PluginDir:     pluginDir,
		Database:      databaseFile,
		buildParams:   make(map[string]interface{}),
	}
}

func (a *LogAgent) WithBuildParameter(key string, value interface{}) *LogAgent {
	a.buildParams[key] = value
	return a
}
