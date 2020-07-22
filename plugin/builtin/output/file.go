package output

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"sync"

	"github.com/observiq/carbon/entry"
	"github.com/observiq/carbon/plugin"
	"github.com/observiq/carbon/plugin/helper"
)

func init() {
	plugin.Register("file_output", func() plugin.Builder { return NewFileOutputConfig("") })
}

func NewFileOutputConfig(pluginID string) *FileOutputConfig {
	return &FileOutputConfig{
		OutputConfig: helper.NewOutputConfig(pluginID, "file_output"),
	}
}

// FileOutputConfig is the configuration of a file output pluginn.
type FileOutputConfig struct {
	helper.OutputConfig `yaml:",inline"`

	Path   string `json:"path" yaml:"path"`
	Format string `json:"format,omitempty" path:"format,omitempty"`
}

// Build will build a file output plugin.
func (c FileOutputConfig) Build(context plugin.BuildContext) (plugin.Operator, error) {
	outputOperator, err := c.OutputConfig.Build(context)
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
		OutputOperator: outputOperator,
		path:           c.Path,
		tmpl:           tmpl,
	}

	return fileOutput, nil
}

// FileOutput is a plugin that writes logs to a file.
type FileOutput struct {
	helper.OutputOperator

	path    string
	tmpl    *template.Template
	encoder *json.Encoder
	file    *os.File
	mux     sync.Mutex
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

// Process will write an entry to the output file.
func (fo *FileOutput) Process(ctx context.Context, entry *entry.Entry) error {
	fo.mux.Lock()
	defer fo.mux.Unlock()

	if fo.tmpl != nil {
		err := fo.tmpl.Execute(fo.file, entry)
		if err != nil {
			return err
		}
	} else {
		err := fo.encoder.Encode(entry)
		if err != nil {
			return err
		}
	}

	return nil
}
