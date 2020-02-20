package plugin

import (
	"encoding/json"
	"fmt"

	bpla "github.com/bluemedora/bplogagent"
)

func init() {
	bpla.RegisterConfig("json", &JSONConfig{})
}

type JSONConfig struct {
	Output string
	Field  string
}

type JSONPlugin struct {
	config JSONConfig
	output EntryProcessor
}

func (p *JSONPlugin) ProcessEntry(entry bpla.Entry) ([]EntryProcessStep, error) {
	message, ok := entry.Record[p.config.Field]
	if !ok {
		return nil, fmt.Errorf("field %s does not exist on the record", p.config.Field)
	}

	messageString, ok := message.(string)
	if !ok {
		return nil, fmt.Errorf("field %s can not be parsed as JSON because it is of type %T", p.config.Field, message)
	}

	var parsedMessage map[string]interface{}
	err := json.Unmarshal([]byte(messageString), &parsedMessage)
	if err != nil {
		return nil, fmt.Errorf("failed to parse field %s as JSON: %w", p.config.Field, err)
	}

	entry.Record[p.config.Field] = parsedMessage
	returnStates := []EntryProcessStep{
		{
			Entry:     entry,
			Processor: p.output,
		},
	}

	return returnStates, nil

}
