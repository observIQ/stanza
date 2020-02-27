package bplogagent

import (
	"fmt"
	"sync"

	"github.com/bluemedora/bplogagent/config"
	"github.com/bluemedora/bplogagent/entry"
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

	plugins, err := pg.BuildPlugins(a.Config.Plugins, a.SugaredLogger)
	if err != nil {
		return err
	}
	a.plugins = plugins

	err = a.startPlugins()
	if err != nil {
		return err
	}

	return nil
}

func (a *LogAgent) Stop() {
	for _, plugin := range a.plugins {
		if source, ok := plugin.(pg.Source); ok {
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

func (a *LogAgent) startPlugins() error {
	closer := &inputChannelCloser{
		waitGroupMap:  make(map[chan<- entry.Entry]*sync.WaitGroup),
		SugaredLogger: a.SugaredLogger,
	}
	defer closer.StartChannelClosers()

	for _, plugin := range a.plugins {
		wg := new(sync.WaitGroup)
		wg.Add(1)
		a.Debugw("Starting plugin", "id", plugin.ID())
		err := plugin.Start(wg)
		if err != nil {
			return fmt.Errorf("failed to start plugin with ID '%s': %s", plugin.ID(), err)
		}

		a.pluginWg.Add(1)
		go func(plugin pg.Plugin, wg *sync.WaitGroup) {
			wg.Wait()
			a.Debugw("Plugin stopped", "id", plugin.ID())
			a.pluginWg.Done()
		}(plugin, wg)

		if outputter, ok := plugin.(pg.Outputter); ok {
			closer.Add(outputter)
			go func(wg *sync.WaitGroup, outputter pg.Outputter) {
				wg.Wait()
				closer.Done(outputter)
			}(wg, outputter)
		}

	}

	return nil
}

type inputChannelCloser struct {
	waitGroupMap map[chan<- entry.Entry]*sync.WaitGroup
	sync.Mutex
	*zap.SugaredLogger
}

func (i *inputChannelCloser) Add(outputter pg.Outputter) {
	i.Lock()
	for _, inputter := range outputter.Outputs() {
		wg, ok := i.waitGroupMap[inputter.Input()]
		if ok {
			wg.Add(1)
		} else {
			newWg := new(sync.WaitGroup)
			newWg.Add(1)
			i.waitGroupMap[inputter.Input()] = newWg
		}
	}
	i.Unlock()
}

func (i *inputChannelCloser) Done(outputter pg.Outputter) {
	i.Lock()
	for _, inputter := range outputter.Outputs() {
		wg, ok := i.waitGroupMap[inputter.Input()]
		if ok {
			wg.Done()
		} else {
			panic("called Done() for a channel that doesn't exist")
		}
	}
	i.Unlock()
}

func (i *inputChannelCloser) StartChannelClosers() {
	i.Lock()
	for channel, waitGroup := range i.waitGroupMap {
		go func(channel chan<- entry.Entry, wg *sync.WaitGroup) {
			wg.Wait()
			close(channel)
		}(channel, waitGroup)
	}
	i.Unlock()
}
