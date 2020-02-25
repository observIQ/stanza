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
	DefaultProcessorConfig `mapstructure:",squash"`
	Field                  string
}

func (c JSONConfig) Build(logger *zap.SugaredLogger) (Plugin, error) {
	plugin := &JSONPlugin{
		DefaultProcessor: c.DefaultProcessorConfig.Build(),
		config:           c,
		SugaredLogger:    logger.With("plugin_type", "json", "plugin_id", c.DefaultProcessorConfig.ID()),
	}

	return plugin, nil
}

type JSONPlugin struct {
	DefaultProcessor
	config JSONConfig
	*zap.SugaredLogger
}

func (s *JSONPlugin) Start(wg *sync.WaitGroup) error {
	go func() {
		defer wg.Done()
		for {
			entry, ok := <-s.Input()
			if !ok {
				return
			}

			newEntry, err := s.processEntry(entry)
			if err != nil {
				s.Warnw("Failed to process entry", "error", err)
				continue
			}

			s.Output() <- newEntry
		}
	}()

	return nil
}

func (p *JSONPlugin) processEntry(entry e.Entry) (e.Entry, error) {
	message, ok := entry.Record[p.config.Field]
	if !ok {
		return e.Entry{}, fmt.Errorf("field %s does not exist on the record", p.config.Field)
	}

	messageString, ok := message.(string)
	if !ok {
		return e.Entry{}, fmt.Errorf("field %s can not be parsed as JSON because it is of type %T", p.config.Field, message)
	}

	var parsedMessage map[string]interface{}
	err := json.Unmarshal([]byte(messageString), &parsedMessage)
	if err != nil {
		return e.Entry{}, fmt.Errorf("failed to parse field %s as JSON: %w", p.config.Field, err)
	}

	entry.Record[p.config.Field] = parsedMessage

	return entry, nil
}
