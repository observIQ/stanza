package plugins

import pg "github.com/bluemedora/bplogagent/plugin"

func NewFakeFileSource() *FileSource {
	out := newFakeNullOutput()
	return &FileSource{
		DefaultPlugin: pg.DefaultPlugin{
			PluginID:   "test",
			PluginType: "file_input",
		},
		DefaultOutputter: pg.DefaultOutputter{
			OutputPlugin: out,
		},
		watchedFiles:       make(map[string]*FileWatcher),
		watchedDirectories: make(map[string]*DirectoryWatcher),
	}
}
