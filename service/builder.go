package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/kardianos/service"
	"github.com/observiq/stanza/v2/operator/helper/persist"
	"github.com/open-telemetry/opentelemetry-log-collection/agent"
	"github.com/open-telemetry/opentelemetry-log-collection/operator"
	"go.uber.org/zap"
)

// AgentServiceBuilder builder for AgentService
type AgentServiceBuilder struct {
	logger       *zap.SugaredLogger
	config       *Config
	databaseFile *string
}

// NewBuilder creates a new AgentServiceBuilder
func NewBuilder() *AgentServiceBuilder {
	return &AgentServiceBuilder{}
}

// WithLogger adds a logger
func (b *AgentServiceBuilder) WithLogger(logger *zap.SugaredLogger) *AgentServiceBuilder {
	b.logger = logger
	return b
}

// WithConfig adds config
func (b *AgentServiceBuilder) WithConfig(config *Config) *AgentServiceBuilder {
	b.config = config
	return b
}

// WithDatabaseFile adds a database file
func (b *AgentServiceBuilder) WithDatabaseFile(datbaseFile string) *AgentServiceBuilder {
	b.databaseFile = &datbaseFile
	return b
}

// Build builds an Agent Service
func (b *AgentServiceBuilder) Build(ctx context.Context) (service.Service, error) {
	logAgent, err := b.buildAgent()
	if err != nil {
		return nil, err
	}

	persister, persisterShutdownFunc, err := b.buildPersister()
	if err != nil {
		return nil, err
	}

	return newAgentService(ctx, logAgent, persister, persisterShutdownFunc, *b.config.PProf)
}

func (b *AgentServiceBuilder) buildAgent() (*agent.LogAgent, error) {
	agentBuilder := agent.NewBuilder(b.logger)

	if b.config == nil {
		return nil, errors.New("config cannot be nil")
	}

	agentBuilder = agentBuilder.WithConfig(&agent.Config{
		Pipeline: b.config.Pipeline,
	})

	logAgent, err := agentBuilder.Build()
	if err != nil {
		return nil, fmt.Errorf("error while constructing agent: %w", err)
	}

	return logAgent, nil
}

func (b *AgentServiceBuilder) buildPersister() (operator.Persister, persist.PersisterShutdownFunc, error) {
	var persister operator.Persister = &persist.NoopPersister{}
	var shutDownFunc persist.PersisterShutdownFunc = persist.NoopShutdownFunc

	// If we have a database file make a bbolt persister
	if b.databaseFile != nil && *b.databaseFile != "" {
		var err error
		persister, shutDownFunc, err = persist.NewBBoltPersister(*b.databaseFile)
		if err != nil {
			return nil, nil, fmt.Errorf("error building bbolt persister: %w", err)
		}
	}

	return persister, shutDownFunc, nil
}
