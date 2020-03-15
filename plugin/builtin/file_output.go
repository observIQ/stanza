package builtin

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"sync"

	"github.com/bluemedora/bplogagent/entry"
	pg "github.com/bluemedora/bplogagent/plugin"
)

func init() {
	pg.RegisterConfig("file_out", &FileOutputConfig{})
}

type FileOutputConfig struct {
	pg.DefaultPluginConfig `mapstructure:",squash" yaml:",inline"`
	Path                   string `yaml:",omitempty"`
	Format                 string `yaml:",omitempty"`
	// TODO file permissions?
}

func (c *FileOutputConfig) Build(buildContext pg.BuildContext) (pg.Plugin, error) {
	defaultPlugin, err := c.DefaultPluginConfig.Build(buildContext.Logger)
	if err != nil {
		return nil, err
	}

	var tmpl *template.Template
	if c.Format != "" {
		tmpl, err = template.New("file").Parse(c.Format)
		if err != nil {
			return nil, err
		}
	}

	if c.Path == "" {
		return nil, fmt.Errorf("must provide a path to output to")
	}
	return &FileOutput{
		DefaultPlugin: defaultPlugin,
		path:          c.Path,
		tmpl:          tmpl,
	}, nil

}

type FileOutput struct {
	pg.DefaultPlugin
	path    string
	tmpl    *template.Template
	encoder *json.Encoder

	file *os.File
	mux  sync.Mutex
}

func (fo *FileOutput) Start() error {
	var err error
	fo.file, err = os.OpenFile(fo.path, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0660)
	if err != nil {
		return err
	}

	fo.encoder = json.NewEncoder(fo.file)

	return nil
}

func (fo *FileOutput) Stop() {
	if fo.file != nil {
		fo.file.Close()
	}
}

func (fo *FileOutput) Input(entry *entry.Entry) error {
	fo.mux.Lock()

	if fo.tmpl != nil {
		err := fo.tmpl.Execute(fo.file, entry)
		if err != nil {
			fo.mux.Unlock() // TODO switch to defer once updated to go 1.14
			return err
		}
	} else {
		err := fo.encoder.Encode(entry)
		if err != nil {
			fo.mux.Unlock()
			return err
		}
	}

	fo.mux.Unlock()
	return nil
}
