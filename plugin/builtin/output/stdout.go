package output

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"sync"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/helper"
)

var Stdout io.Writer = os.Stdout

func init() {
	plugin.Register("stdout", &StdoutConfig{})
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
