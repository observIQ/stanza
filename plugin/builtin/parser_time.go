package builtin

import (
	"fmt"
	"time"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/helper"
	"go.uber.org/zap"
)

func init() {
	plugin.Register("time_parser", &TimeParserConfig{})
}

// TimeParserConfig is the configuration of a time parser plugin.
// TODO split this into a "parser" and a "promoter" plugin?
type TimeParserConfig struct {
	helper.BasicPluginConfig      `mapstructure:",squash" yaml:",inline"`
	helper.BasicTransformerConfig `mapstructure:",squash" yaml:",inline"`

	Field  entry.FieldSelector
	Layout string
}

// Build will build a time parser plugin.
func (c TimeParserConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	basicPlugin, err := c.BasicPluginConfig.Build(context.Logger)
	if err != nil {
		return nil, err
	}

	basicTransformer, err := c.BasicTransformerConfig.Build()
	if err != nil {
		return nil, err
	}

	if c.Field == nil {
		return nil, fmt.Errorf("missing required field 'field'")
	}

	if c.Layout == "" {
		return nil, fmt.Errorf("missing required field 'layout'")
	}

	timeParserPlugin := &TimeParser{
		BasicPlugin:      basicPlugin,
		BasicTransformer: basicTransformer,

		field:  c.Field,
		layout: c.Layout,
	}

	return timeParserPlugin, nil
}

// TimeParser is a plugin that parses time from a field.
type TimeParser struct {
	helper.BasicPlugin
	helper.BasicLifecycle
	helper.BasicTransformer

	field  entry.FieldSelector
	layout string
}

// Process will parse time and send the entry to the next plugin.
func (p *TimeParser) Process(entry *entry.Entry) error {
	newEntry, err := p.parseTime(entry)
	if err != nil {
		p.Warnw("Failed to parse time", zap.Error(err), "entry", entry)
		return p.Output.Process(entry)
	}

	return p.Output.Process(newEntry)
}

// Parse time will parse time and create a new entry.
func (p *TimeParser) parseTime(entry *entry.Entry) (*entry.Entry, error) {
	message, ok := entry.Get(p.field)
	if !ok {
		return nil, fmt.Errorf("field '%s' does not exist", p.field)
	}

	// TODO support bytes?
	messageString, ok := message.(string)
	if !ok {
		return nil, fmt.Errorf("field '%s' can not be parsed with regex because it is of type %T", p.field, message)
	}

	time, err := time.Parse(p.layout, messageString)
	if err != nil {
		return nil, fmt.Errorf("parsing time: %s", err)
	}

	entry.Timestamp = time
	entry.Delete(p.field)

	return entry, nil
}
