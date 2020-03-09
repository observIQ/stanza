package plugins

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	pg "github.com/bluemedora/bplogagent/plugin"
	// using Docker's filenotify so we can fall back to polling
	// for envs where notify isn't available
)

func init() {
	pg.RegisterConfig("file", &FileSourceConfig{})
}

type FileSourceConfig struct {
	pg.DefaultPluginConfig
	pg.DefaultOutputterConfig

	Include              []string
	Exclude              []string
	FallbackPollInterval float64
}

func (c FileSourceConfig) Build(buildContext pg.BuildContext) (pg.Plugin, error) {
	defaultPlugin, err := c.DefaultPluginConfig.Build(buildContext.Logger)
	if err != nil {
		return nil, fmt.Errorf("build default plugin: %s", err)
	}

	defaultOutputter, err := c.DefaultOutputterConfig.Build(buildContext.Plugins)
	if err != nil {
		return nil, fmt.Errorf("build default outputter: %s", err)
	}

	for _, include := range c.Include {
		_, err := filepath.Match(include, "")
		if err != nil {
			return nil, fmt.Errorf("parse include glob: %s", err)
		}
	}

	for _, exclude := range c.Exclude {
		_, err := filepath.Match(exclude, "")
		if err != nil {
			return nil, fmt.Errorf("parse exclude glob: %s", err)
		}
	}

	plugin := &FileSource{
		DefaultPlugin:    defaultPlugin,
		DefaultOutputter: defaultOutputter,
	}

	return plugin, nil
}

type FileSource struct {
	pg.DefaultPlugin
	pg.DefaultOutputter

	Include []string
	Exclude []string

	wg                 *sync.WaitGroup
	cancel             context.CancelFunc
	watchedFiles       map[string]*FileWatcher
	fmux               sync.Mutex
	watchedDirectories map[string]*DirectoryWatcher
	dmux               sync.Mutex
}

func (f *FileSource) Start() error {
	ctx, cancel := context.WithCancel(context.Background())
	f.cancel = cancel
	f.wg = &sync.WaitGroup{}

	f.wg.Add(1)
	go func() {
		defer f.wg.Done()
		globTicker := time.NewTicker(time.Second) // TODO tune this param and make it configurable
		for {
			select {
			case <-ctx.Done():
				// TODO
			case <-globTicker.C:
				f.updateFiles(ctx)
			}
		}
	}()

	return nil
}

func (f *FileSource) Stop() {
	f.cancel()
	f.wg.Wait()
}

func (f *FileSource) updateFiles(ctx context.Context) {
	for _, includePattern := range f.Include {
		matches, _ := filepath.Glob(includePattern)
		for _, path := range matches {
			if !f.isExcluded(path) {
				f.tryWatchFile(ctx, path)
			}
		}
	}
}

func (f *FileSource) isExcluded(path string) bool {
	for _, excludePattern := range f.Exclude {
		// error already checked in build step
		if exclude, _ := filepath.Match(excludePattern, path); exclude {
			return true
		}
	}

	return false
}

func (f *FileSource) tryWatchFile(ctx context.Context, path string) {
	f.fmux.Lock()
	defer f.fmux.Unlock()

	f.tryWatchDirectory(ctx, path)

	_, ok := f.watchedFiles[path]
	if ok {
		return
	}

	watcher, err := NewFileWatcher(path)
	if err != nil {
		println("Creating file watcher: ", err) // TODO
	}

	f.watchedFiles[path] = watcher
	f.Infow("Watching file", "path", path)

	f.wg.Add(1)
	go func() {
		defer f.wg.Done()
		err := watcher.Watch(ctx)
		if err != nil {
			f.Infow("Stopped watching file", "path", path)
			f.removeFileWatcher(path)
		}
	}()
}

func (f *FileSource) tryWatchDirectory(ctx context.Context, path string) {
	f.dmux.Lock()
	defer f.dmux.Unlock()

	_, ok := f.watchedDirectories[path]
	if ok {
		return
	}

	watcher, err := NewDirectoryWatcher(path, func(path string) { f.tryWatchFile(ctx, path) })
	if err != nil {
		println("Creating directory watcher: ", err) // TODO
	}

	f.watchedDirectories[path] = watcher
	f.Infow("Watching directory", "path", path)

	f.wg.Add(1)
	go func() {
		defer f.wg.Done()
		err := watcher.Watch(ctx)
		if err != nil {
			f.Infow("Stopped watching directory", "path", path)
			f.removeDirectoryWatcher(path)
		}
	}()
}

func (f *FileSource) removeDirectoryWatcher(path string) {
	f.dmux.Lock()
	defer f.dmux.Unlock()

	delete(f.watchedDirectories, path)
}

func (f *FileSource) removeFileWatcher(path string) {
	f.fmux.Lock()
	defer f.fmux.Unlock()

	delete(f.watchedFiles, path)
}
