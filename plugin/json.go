package plugin

import (
	"encoding/json"
	"fmt"

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

	return &SimpleProcessorAdapter{plugin}, nil
}

type JSONPlugin struct {
	DefaultProcessor
	config JSONConfig
	*zap.SugaredLogger
}

func (p *JSONPlugin) ProcessEntry(entry e.Entry) (e.Entry, error) {
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

func (p *JSONPlugin) Logger() *zap.SugaredLogger {
	return p.SugaredLogger
}
