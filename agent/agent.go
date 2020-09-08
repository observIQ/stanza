package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/observiq/stanza/errors"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/pipeline"
	"go.etcd.io/bbolt"
	"go.uber.org/zap"
)

// LogAgent is an entity that handles log monitoring.
type LogAgent struct {
	database operator.Database
	pipeline *pipeline.Pipeline

	startOnce sync.Once
	stopOnce  sync.Once

	*zap.SugaredLogger
}

// Start will start the log monitoring process.
func (a *LogAgent) Start() (err error) {
	a.startOnce.Do(func() {
		err = a.pipeline.Start()
		if err != nil {
			return
		}
		a.Info("Agent started")
	})
	return
}

// Stop will stop the log monitoring process.
func (a *LogAgent) Stop() {
	a.stopOnce.Do(func() {
		a.pipeline.Stop()
		a.database.Close()
		a.Info("Agent stopped")
	})
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

// LogAgentBuilder is a construct used to build a log agent
type LogAgentBuilder struct {
	cfg           *Config
	logger        *zap.SugaredLogger
	pluginDir     string
	databaseFile  string
	defaultOutput operator.Operator
}

// NewBuilder creates a new LogAgentBuilder
func NewBuilder(cfg *Config, logger *zap.SugaredLogger) *LogAgentBuilder {
	return &LogAgentBuilder{
		cfg:    cfg,
		logger: logger,
	}
}

// WithPluginDir adds the specified plugin directory when building a log agent
func (b *LogAgentBuilder) WithPluginDir(pluginDir string) *LogAgentBuilder {
	b.pluginDir = pluginDir
	return b
}

// WithDatabaseFile adds the specified database file when building a log agent
func (b *LogAgentBuilder) WithDatabaseFile(databaseFile string) *LogAgentBuilder {
	b.databaseFile = databaseFile
	return b
}

// WithDefaultOutput adds a default output when building a log agent
func (b *LogAgentBuilder) WithDefaultOutput(defaultOutput operator.Operator) *LogAgentBuilder {
	b.defaultOutput = defaultOutput
	return b
}

// Build will build a new log agent using the values defined on the builder
func (b *LogAgentBuilder) Build() (*LogAgent, error) {
	database, err := OpenDatabase(b.databaseFile)
	if err != nil {
		return nil, errors.Wrap(err, "open database")
	}

	registry, err := operator.NewPluginRegistry(b.pluginDir)
	if err != nil {
		return nil, errors.Wrap(err, "load plugin registry")
	}

	buildContext := operator.BuildContext{
		Logger:         b.logger,
		PluginRegistry: registry,
		Database:       database,
	}

	pipeline, err := b.cfg.Pipeline.BuildPipeline(buildContext, b.defaultOutput)
	if err != nil {
		return nil, err
	}

	return &LogAgent{
		pipeline:      pipeline,
		database:      database,
		SugaredLogger: b.logger,
	}, nil
}
