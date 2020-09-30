package agent

import (
	"github.com/observiq/stanza/database"
	"github.com/observiq/stanza/errors"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/plugin"
	"go.uber.org/zap"
)

// LogAgentBuilder is a construct used to build a log agent
type LogAgentBuilder struct {
	configFiles   []string
	config        *Config
	logger        *zap.SugaredLogger
	pluginDir     string
	databaseFile  string
	defaultOutput operator.Operator
}

// NewBuilder creates a new LogAgentBuilder
func NewBuilder(logger *zap.SugaredLogger) *LogAgentBuilder {
	return &LogAgentBuilder{
		logger: logger,
	}
}

// WithPluginDir adds the specified plugin directory when building a log agent
func (b *LogAgentBuilder) WithPluginDir(pluginDir string) *LogAgentBuilder {
	b.pluginDir = pluginDir
	return b
}

// WithPluginDir adds the specified plugin directory when building a log agent
func (b *LogAgentBuilder) WithConfigFiles(files []string) *LogAgentBuilder {
	b.configFiles = files
	return b
}

func (b *LogAgentBuilder) WithConfig(cfg *Config) *LogAgentBuilder {
	b.config = cfg
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

	if b.pluginDir != "" {
		if err := plugin.RegisterPlugins(b.pluginDir, operator.DefaultRegistry); err != nil {
			return nil, err
		}
	}

	if b.config != nil && len(b.configFiles) > 0 {
		return nil, errors.NewError("agent can be built WithConfig or WithConfigFiles, but not both", "")
	} else if len(b.configFiles) > 0 {
		b.config, err = NewConfigFromGlobs(b.configFiles)
		if err != nil {
			return nil, errors.Wrap(err, "read configs from globs")
		}
	}

	buildContext := operator.BuildContext{
		Logger:    b.logger,
		Database:  db,
		Namespace: "$",
	}

	pipeline, err := b.config.Pipeline.BuildPipeline(buildContext)
	if err != nil {
		return nil, err
	}

	return &LogAgent{
		pipeline:      pipeline,
		database:      db,
		SugaredLogger: b.logger,
	}, nil
}
