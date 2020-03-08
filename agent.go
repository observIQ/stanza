package bplogagent

import (
	"github.com/bluemedora/bplogagent/bundle"
	"github.com/bluemedora/bplogagent/config"
	pg "github.com/bluemedora/bplogagent/plugin"
	_ "github.com/bluemedora/bplogagent/plugin/plugins" // register plugins
	"go.uber.org/zap"
)

func NewLogAgent(cfg config.Config, logger *zap.SugaredLogger) *LogAgent {
	return &LogAgent{
		Config:        cfg,
		SugaredLogger: logger,
		started:       make(chan struct{}, 1),
	}
}

type LogAgent struct {
	Config config.Config

	plugins *pg.PluginGraph
	started chan struct{}
	*zap.SugaredLogger
}

func (a *LogAgent) Start() error {
	// TODO protect against multiple starts
	configGraph, err := pg.NewPluginConfigGraph(a.Config.Plugins)
	if err != nil {
		return err
	}

	bundles := bundle.GetBundleDefinitions(a.Config.BundlePath, a.SugaredLogger)
	buildContext := pg.BuildContext{
		Logger:  a.SugaredLogger,
		Plugins: make(map[pg.PluginID]pg.Plugin),
		Bundles: bundles,
	}
	a.plugins, err = configGraph.Build(buildContext)
	if err != nil {
		return err
	}
	err = a.plugins.Start()
	if err != nil {
		return err
	}

	return nil
}

func (a *LogAgent) Stop() {
	a.Info("Stopping plugins")
	a.plugins.Stop()

	a.plugins = nil
	<-a.started
	a.Info("Log agent stopped cleanly")
}

func (a *LogAgent) Status() struct{} {
	return struct{}{}
}
