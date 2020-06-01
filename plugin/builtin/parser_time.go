package builtin

import (
	"fmt"
	"time"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/helper"
)

func init() {
	plugin.Register("time_parser", &TimeParserConfig{})
}

// TimeParserConfig is the configuration of a time parser plugin.
type TimeParserConfig struct {
	helper.ParserConfig `mapstructure:",squash" yaml:",inline"`

	Layout string `mapstructure:"layout" json:"layout" yaml:"layout"`
}

// Build will build a time parser plugin.
func (c TimeParserConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	parserPlugin, err := c.ParserConfig.Build(context)
	if err != nil {
		return nil, err
	}

	timeParser := &TimeParser{
		ParserPlugin: parserPlugin,
		layout:       c.Layout,
	}

	return timeParser, nil
}

// TimeParser is a plugin that parses time from an entry.
type TimeParser struct {
	helper.ParserPlugin
	layout string
}

// Process will parse time from an entry.
func (t *TimeParser) Process(entry *entry.Entry) error {
	return t.ProcessWith(entry, t.parse)
}

// Parse will parse a value as a time.
func (t *TimeParser) parse(value interface{}) (interface{}, error) {
	switch v := value.(type) {
	case string:
		return time.Parse(t.layout, v)
	case []byte:
		return time.Parse(t.layout, string(v))
	default:
		return nil, fmt.Errorf("type %T cannot be parsed as a time", value)
	}
}
