package plugin

import (
	"encoding/json"
	"fmt"
	"sync"

	e "github.com/bluemedora/bplogagent/entry"
	"go.uber.org/zap"
)

func init() {
	RegisterConfig("json", &JSONConfig{})
}

type JSONConfig struct {
	DefaultPluginConfig    `mapstructure:",squash"`
	DefaultOutputterConfig `mapstructure:",squash"`
	DefaultInputterConfig  `mapstructure:",squash"`

	// TODO design these params better
	Field            string
	DestinationField string
}

func (c JSONConfig) Build(plugins map[PluginID]Plugin, logger *zap.SugaredLogger) (Plugin, error) {
	defaultPlugin, err := c.DefaultPluginConfig.Build(logger)
	if err != nil {
		return nil, fmt.Errorf("failed to build default plugin: %s", err)
	}

	defaultInputter, err := c.DefaultInputterConfig.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build default inputter: %s", err)
	}

	defaultOutputter, err := c.DefaultOutputterConfig.Build(plugins)
	if err != nil {
		return nil, fmt.Errorf("failed to build default outputter: %s", err)
	}

	if c.Field == "" {
		return nil, fmt.Errorf("missing required field 'field'")
	}

	plugin := &JSONPlugin{
		DefaultPlugin:    defaultPlugin,
		DefaultInputter:  defaultInputter,
		DefaultOutputter: defaultOutputter,

		field:            c.Field,
		destinationField: c.DestinationField,
	}

	return plugin, nil
}

type JSONPlugin struct {
	DefaultPlugin
	DefaultOutputter
	DefaultInputter

	field            string
	destinationField string
}

func (p *JSONPlugin) Start(wg *sync.WaitGroup) error {
	go func() {
		defer wg.Done()
		for {
			entry, ok := <-p.Input()
			if !ok {
				return
			}

			newEntry, err := p.processEntry(entry)
			if err != nil {
				// TODO better error handling
				p.Warnw("Failed to process entry", "error", err)
				continue
			}

			p.Output() <- newEntry
		}
	}()

	return nil
}

func (p *JSONPlugin) processEntry(entry e.Entry) (e.Entry, error) {
	message, ok := entry.Record[p.field]
	if !ok {
		return e.Entry{}, fmt.Errorf("field '%s' does not exist on the record", p.field)
	}

	messageString, ok := message.(string)
	if !ok {
		return e.Entry{}, fmt.Errorf("field '%s' can not be parsed as JSON because it is of type %T", p.field, message)
	}

	// TODO consider using faster json decoder (fastjson?)
	var parsedMessage map[string]interface{}
	err := json.Unmarshal([]byte(messageString), &parsedMessage)
	if err != nil {
		return e.Entry{}, fmt.Errorf("failed to parse field %s as JSON: %w", p.field, err)
	}

	if p.destinationField == "" {
		entry.Record[p.field] = parsedMessage
	} else {
		entry.Record[p.destinationField] = parsedMessage
	}

	return entry, nil
}
