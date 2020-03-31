package builtin

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"sync"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/base"
)

func init() {
	plugin.Register("file_output", &FileOutputConfig{})
}

// FileOutputConfig is the configuration of a file output pluginn.
type FileOutputConfig struct {
	base.OutputConfig `mapstructure:",squash" yaml:",inline"`
	Path              string `yaml:",omitempty"`
	Format            string `yaml:",omitempty"`
	// TODO file permissions?
}

// Build will build a file output plugin.
func (c *FileOutputConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	outputPlugin, err := c.OutputConfig.Build(context)
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

	fileOutput := &FileOutput{
		OutputPlugin: outputPlugin,
		path:         c.Path,
		tmpl:         tmpl,
	}

	return fileOutput, nil
}

// FileOutput is a plugin that writes logs to a file.
type FileOutput struct {
	base.OutputPlugin
	path    string
	tmpl    *template.Template
	encoder *json.Encoder

	file *os.File
	mux  sync.Mutex
}

// Start will open the output file.
func (fo *FileOutput) Start() error {
	var err error
	fo.file, err = os.OpenFile(fo.path, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0660)
	if err != nil {
		return err
	}

	fo.encoder = json.NewEncoder(fo.file)

	return nil
}

// Stop will close the output file.
func (fo *FileOutput) Stop() error {
	if fo.file != nil {
		fo.file.Close()
	}
	return nil
}

// Consume will write an entry to the output file.
func (fo *FileOutput) Consume(entry *entry.Entry) error {
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
