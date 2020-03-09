package plugins

import (
	"testing"

	pg "github.com/bluemedora/bplogagent/plugin"
	"github.com/stretchr/testify/assert"
)

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

func TestFileSource(t *testing.T) {
	source := NewFakeFileSource()
	err := source.Start()
	assert.NoError(t, err)
	source.Stop()
}
