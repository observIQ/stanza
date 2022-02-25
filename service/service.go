package service

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/kardianos/service"
	"github.com/observiq/stanza/v2/operator/helper/persist"
	"github.com/open-telemetry/opentelemetry-log-collection/agent"
	"github.com/open-telemetry/opentelemetry-log-collection/operator"
	"go.uber.org/zap"
)

// AgentService is a service that runs the stanza agent.
type AgentService struct {
	cancel                context.CancelFunc
	agent                 *agent.LogAgent
	persister             operator.Persister
	persisterShutdownFunc persist.PersisterShutdownFunc
	pprofProfiler         *pprofProfiler
}

// Start will start the stanza agent.
func (a *AgentService) Start(_ service.Service) error {
	a.agent.Info("Starting stanza agent")
	if err := a.agent.Start(a.persister); err != nil {
		a.agent.Errorw("Failed to start stanza agent", zap.Error(err))
		a.cancel()
		return nil
	}

	// This will start the profiler, if it was enabled in the user config
	if err := a.pprofProfiler.Start(); err != nil {
		a.agent.Errorw("Failed to start pprof service", zap.Error(err))
		a.cancel()
		return nil
	}

	a.agent.Info("Stanza agent started")
	return nil
}

// Stop will stop the stanza agent.
func (a *AgentService) Stop(_ service.Service) error {
	defer a.cancel()
	defer a.persisterShutdownFunc()
	a.agent.Info("Stopping stanza agent")
	if err := a.agent.Stop(); err != nil {
		a.agent.Errorw("Failed to stop stanza agent gracefully", zap.Error(err))
	}

	a.pprofProfiler.Stop()

	a.agent.Info("Stanza agent stopped")
	return nil
}

// newAgentService creates a new agent service with the provided agent.
func newAgentService(ctx context.Context, agent *agent.LogAgent,
	persister operator.Persister, persisterShutdownFunc persist.PersisterShutdownFunc, pprofConfig PProfConfig) (service.Service, error) {
	// Create a context for this service based on the passed in context
	serviceCtx, serviceCancel := context.WithCancel(ctx)

	agentService := &AgentService{
		cancel:                serviceCancel,
		agent:                 agent,
		persister:             persister,
		persisterShutdownFunc: persisterShutdownFunc,
		pprofProfiler:         newPProfProfiler(serviceCtx, agent.SugaredLogger, pprofConfig),
	}
	config := &service.Config{
		Name:        "stanza",
		DisplayName: "Stanza Log Agent",
		Description: "Monitors and processes log entries",
		Option: service.KeyValue{
			"RunWait": func() {
				// Create a child context off of signal notify and block on it
				signalCtx, cancel := signal.NotifyContext(serviceCtx, syscall.SIGTERM, os.Interrupt)
				defer cancel()
				<-signalCtx.Done()
			},
		},
	}

	service, err := service.New(agentService, config)
	if err != nil {
		return nil, err
	}

	return service, nil
}