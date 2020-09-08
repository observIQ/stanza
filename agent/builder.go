package agent

import (
	"github.com/observiq/stanza/database"
	"github.com/observiq/stanza/errors"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/plugin"
	"go.uber.org/zap"
)

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
	db, err := database.OpenDatabase(b.databaseFile)
	if err != nil {
		return nil, errors.Wrap(err, "open database")
	}

	registry, err := plugin.NewPluginRegistry(b.pluginDir)
	if err != nil {
		return nil, errors.Wrap(err, "load plugin registry")
	}

	buildContext := operator.BuildContext{
		Logger:   b.logger,
		Database: db,
	}

	pipeline, err := b.cfg.Pipeline.BuildPipeline(buildContext, registry, b.defaultOutput)
	if err != nil {
		return nil, err
	}

	return &LogAgent{
		pipeline:      pipeline,
		database:      db,
		SugaredLogger: b.logger,
	}, nil
}
