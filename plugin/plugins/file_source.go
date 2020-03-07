package plugins

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	pg "github.com/bluemedora/bplogagent/plugin"
	"github.com/fsnotify/fsnotify"

	// using Docker's filenotify so we can fall back to polling
	// for envs where notify isn't available
	"github.com/docker/docker/pkg/filenotify"
)

func init() {
	pg.RegisterConfig("file", &FileSourceConfig{})
}

type FileSourceConfig struct {
	pg.DefaultPluginConfig
	pg.DefaultOutputterConfig

	// TODO allow multiple glob patterns?
	Path string
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

	_, err = filepath.Glob(c.Path)
	if err != nil {
		return nil, fmt.Errorf("parse glob: %s", err)
	}

	plugin := &FileSource{
		DefaultPlugin:    defaultPlugin,
		DefaultOutputter: defaultOutputter,
		Path:             c.Path,
	}

	return plugin, nil
}

type FileSource struct {
	pg.DefaultPlugin
	pg.DefaultOutputter

	Path string
}

func (f *FileSource) Start(wg *sync.WaitGroup) error {

	ready := make(chan error)
	go func() {
		defer wg.Done()
		paths, err := filepath.Glob(f.Path)
		if err != nil {
			ready <- err
		}

		watcher, err := filenotify.New()
		if err != nil {
			ready <- err
		}

		for _, path := range paths {
			err := watcher.Add(path)
			if err != nil {

			}
		}

	}()

	return <-ready
}

// FileWatcher is a wrapper around `fsnotify` that periodically polls
// to mitigate issues with filesystems that don't support notify events
type FileWatcher struct {
	path         string
	file         *os.File
	watcher      *fsnotify.Watcher
	offset       int
	pollInterval time.Duration
}

func NewFileWatcher(path string) (*FileWatcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("creating fsnotify watcher: %s", err)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("determining absolute path: %s", err)
	}

	return &FileWatcher{
		path:         absPath,
		watcher:      w,
		pollInterval: 3 * time.Second,
	}, err
}

func (w *FileWatcher) Watch() error {
	for {
		// TODO actually test all these cases
		// TODO actually test all these cases on every OS we support
		// TODO actually test all these cases on weird filesystems (NFS, FUSE, etc)

		// TODO reuse the timer? but be careful about draining -- see timer.Reset() docs
		timer := time.NewTimer(w.pollInterval)

		select {
		case event, ok := <-w.watcher.Events:
			// Stop the timer so it can be garbage collected before it fires
			timer.Stop()

			// Watcher closed
			if !ok {
				return nil
			}

			// File touched
			if event.Op&fsnotify.Write > 0 {
				w.Read()
			}

			// File touched
			if event.Op&fsnotify.Create > 0 {
				w.offset = 0
				w.Read()
			}

			// File removed
			if event.Op&fsnotify.Remove == fsnotify.Remove {
				return nil
			}
		case <-timer.C:
			// Try to read file anyways
			w.Read()
		case err := <-w.watcher.Errors:
			timer.Stop()
			// TODO should we exit?
			return err
		}

	}
}

func (w *FileWatcher) Close() {
	w.watcher.Close()
}

func (w *FileWatcher) Read() {}
