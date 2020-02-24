package bplogagent

import (
	"fmt"
	"sync"

	"github.com/bluemedora/bplogagent/config"
	"github.com/bluemedora/bplogagent/entry"
	pg "github.com/bluemedora/bplogagent/plugin"
	"go.uber.org/zap"
)

type LogAgent struct {
	Config config.Config

	plugins []pg.Plugin
	wg      *sync.WaitGroup
	*zap.SugaredLogger
}

func (a *LogAgent) Start() error {
	// TODO protect against duplicate starts
	a.Info("Starting log collector")
	a.wg = new(sync.WaitGroup)

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
	a.wg.Wait()
	a.Info("Log agent stopped cleanly")
}

func (a *LogAgent) Status() struct{} {
	return struct{}{}
}

func (a *LogAgent) startPlugins() error {
	inputChannelWaitGroups := &inputChannelCloser{
		waitGroupMap: make(map[chan<- entry.Entry]*sync.WaitGroup),
	}
	defer inputChannelWaitGroups.StartChannelClosers()

	for _, plugin := range a.plugins {
		wg := new(sync.WaitGroup)
		wg.Add(1)
		err := plugin.Start(wg)
		if err != nil {
			return fmt.Errorf("failed to start plugin with ID '%s': %s", plugin.ID(), err)
		}

		if outputter, ok := plugin.(pg.Outputter); ok {
			a.wg.Add(1)
			inputChannelWaitGroups.Add(outputter.Outputs())
			go func() {
				wg.Wait()
				a.wg.Done()
				inputChannelWaitGroups.Done(outputter.Outputs())
			}()
		}
	}

	return nil
}

type inputChannelCloser struct {
	waitGroupMap map[chan<- entry.Entry]*sync.WaitGroup
	sync.Mutex
}

func (i *inputChannelCloser) Add(channels []chan<- entry.Entry) {
	i.Lock()
	for _, channel := range channels {
		wg, ok := i.waitGroupMap[channel]
		if ok {
			wg.Add(1)
		} else {
			newWg := new(sync.WaitGroup)
			newWg.Add(1)
			i.waitGroupMap[channel] = newWg
		}
	}
	i.Unlock()
}

func (i *inputChannelCloser) Done(channels []chan<- entry.Entry) {
	i.Lock()
	for _, channel := range channels {
		wg, ok := i.waitGroupMap[channel]
		if ok {
			wg.Done()
		}
	}
	i.Unlock()
}

func (i *inputChannelCloser) StartChannelClosers() {
	i.Lock()
	for channel, waitGroup := range i.waitGroupMap {
		go func(channel chan<- entry.Entry, waitGroup *sync.WaitGroup) {
			waitGroup.Wait()
			close(channel)
		}(channel, waitGroup)
	}
	i.Unlock()
}
