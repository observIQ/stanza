package output

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"sync"

	"github.com/observiq/carbon/entry"
	"github.com/observiq/carbon/plugin"
	"github.com/observiq/carbon/plugin/helper"
)

// Stdout is a global handle to standard output
var Stdout io.Writer = os.Stdout

func init() {
	plugin.Register("stdout", func() plugin.Builder { return NewStdoutConfig("") })
}

func NewStdoutConfig(pluginID string) *StdoutConfig {
	return &StdoutConfig{
		OutputConfig: helper.NewOutputConfig(pluginID, "stdout"),
	}
}

// StdoutConfig is the configuration of the Stdout plugin
type StdoutConfig struct {
	helper.OutputConfig `yaml:",inline"`
}

// Build will build a stdout plugin.
func (c StdoutConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	outputPlugin, err := c.OutputConfig.Build(context)
	if err != nil {
		return nil, err
	}

	return &StdoutPlugin{
		OutputPlugin: outputPlugin,
		encoder:      json.NewEncoder(Stdout),
	}, nil
}

// StdoutPlugin is a plugin that logs entries using stdout.
type StdoutPlugin struct {
	helper.OutputPlugin
	encoder *json.Encoder
	mux     sync.Mutex
}

// Process will log entries received.
func (o *StdoutPlugin) Process(ctx context.Context, entry *entry.Entry) error {
	o.mux.Lock()
	err := o.encoder.Encode(entry)
	if err != nil {
		o.mux.Unlock()
		o.Errorf("Failed to process entry: %s, $s", err, entry.Record)
		return err
	}
	o.mux.Unlock()
	return nil
}
