package bplogagent

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/bluemedora/bplogagent/bundle"
	"github.com/bluemedora/bplogagent/config"
	pg "github.com/bluemedora/bplogagent/plugin"
	_ "github.com/bluemedora/bplogagent/plugin/builtin" // register plugins
	"go.etcd.io/bbolt"
	"go.uber.org/zap"
)

func NewLogAgent(cfg *config.Config, logger *zap.SugaredLogger) *LogAgent {
	return &LogAgent{
		Config:        cfg,
		SugaredLogger: logger,
		started:       make(chan struct{}, 1),
	}
}

type LogAgent struct {
	Config *config.Config

	plugins *pg.PluginGraph
	started chan struct{}
	*zap.SugaredLogger
	closeDB func()
}

func (a *LogAgent) Start() error {
	// TODO protect against multiple starts
	configGraph, err := pg.NewPluginConfigGraph(a.Config.Plugins)
	if err != nil {
		return err
	}

	bundles := bundle.GetBundleDefinitions(a.Config.BundlePath, a.SugaredLogger)
	dbFile := func() string {
		if a.Config.DatabaseFile == "" {
			dir, err := os.UserCacheDir()
			if err != nil {
				return filepath.Join(".", "bplogagent.db")
			}
			return filepath.Join(dir, "bplogagent.db")
		}

		return a.Config.DatabaseFile
	}()
	db, err := bbolt.Open(dbFile, 0666, &bbolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return fmt.Errorf("open database: %s", err)
	}
	a.closeDB = func() {
		db.Close()
	}

	buildContext := pg.BuildContext{
		Logger:   a.SugaredLogger,
		Plugins:  make(map[pg.PluginID]pg.Plugin),
		Bundles:  bundles,
		Database: db,
	}

	a.plugins, err = configGraph.Build(buildContext)
	if err != nil {
		return fmt.Errorf("build plugin graph: %s", err)
	}

	dotGraph, err := a.plugins.MarshalDot()
	if err != nil {
		a.Warnw("Failed to marshal plugin graph as dot", "error", err)
	} else {
		a.Infof("Plugin graph:\n%s", dotGraph)
	}

	err = a.plugins.Start()
	if err != nil {
		return err
	}
	a.Info("Started plugins")

	return nil
}

func (a *LogAgent) Stop() {
	a.Info("Stopping plugins")
	a.plugins.Stop()

	a.plugins = nil
	if a.closeDB != nil {
		a.closeDB()
	}
	a.Info("Log agent stopped cleanly")
}

func (a *LogAgent) Status() struct{} {
	return struct{}{}
}
