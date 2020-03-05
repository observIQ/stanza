package bplogagent

import (
	"fmt"
	"sync"

	"github.com/bluemedora/bplogagent/bundle"
	"github.com/bluemedora/bplogagent/config"
	pg "github.com/bluemedora/bplogagent/plugin"
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

	plugins  []pg.Plugin
	pluginWg *sync.WaitGroup
	started  chan struct{}
	*zap.SugaredLogger
}

func (a *LogAgent) Start() error {
	// TODO abstract this?
	select {
	case a.started <- struct{}{}:
	default:
		return fmt.Errorf("log agent is already running")
	}

	a.Info("Starting log collector")
	a.pluginWg = new(sync.WaitGroup)

	bundles := bundle.GetBundleDefinitions(a.Config.BundlePath, a.SugaredLogger)

	buildContext := pg.BuildContext{
		Logger:  a.SugaredLogger,
		Plugins: make(map[pg.PluginID]pg.Plugin),
		Bundles: bundles,
	}
	plugins, err := pg.BuildPlugins(a.Config.Plugins, buildContext)
	if err != nil {
		return err
	}
	a.plugins = plugins

	err = pg.StartPlugins(a.plugins, a.pluginWg, a.SugaredLogger)
	if err != nil {
		return err
	}

	return nil
}

func (a *LogAgent) Stop() {
	for _, plugin := range a.plugins {
		if source, ok := plugin.(pg.Stopper); ok {
			source.Stop()
		}
	}
	a.Info("Waiting for plugins to exit cleanly")
	a.pluginWg.Wait()
	a.plugins = nil
	a.pluginWg = nil
	<-a.started
	a.Info("Log agent stopped cleanly")
}

func (a *LogAgent) Status() struct{} {
	return struct{}{}
}
