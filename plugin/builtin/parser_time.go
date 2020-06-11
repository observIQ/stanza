package builtin

import (
	"context"
	"fmt"
	"time"

	strptime "github.com/bluemedora/ctimefmt"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/errors"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/helper"
)

// Valid layout flavors
const strptimeKey = "strptime"
const gotimeKey = "gotime"

func init() {
	plugin.Register("time_parser", &TimeParserConfig{})
}

// TimeParserConfig is the configuration of a time parser plugin.
type TimeParserConfig struct {
	helper.TransformerConfig `yaml:",inline"`

	ParseFrom    entry.Field `json:"parse_from,omitempty" yaml:"parse_from,omitempty"`
	Layout       string      `json:"layout,omitempty" yaml:"layout,omitempty"`
	LayoutFlavor string      `json:"layout_flavor,omitempty" yaml:"layout_flavor,omitempty"`
}

// Build will build a time parser plugin.
func (c TimeParserConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	transformerPlugin, err := c.TransformerConfig.Build(context)
	if err != nil {
		return nil, err
	}

	switch c.LayoutFlavor {
	case strptimeKey: // ok
	case gotimeKey: // ok
	case "":
		c.LayoutFlavor = strptimeKey
	default:
		return nil, errors.NewError(fmt.Sprintf("Unsupported layout_flavor %s", c.LayoutFlavor), "Valid values are 'strptime' or 'gotime'",
			"plugin_id", c.PluginID,
			"plugin_type", c.PluginType,
		)
	}

	if c.Layout == "" {
		return nil, errors.NewError("Missing required configuration parameter `layout`", "",
			"plugin_id", c.PluginID,
			"plugin_type", c.PluginType,
		)
	}

	timeParser := &TimeParser{
		TransformerPlugin: transformerPlugin,
		ParseFrom:         c.ParseFrom,
		LayoutFlavor:      c.LayoutFlavor,
		Layout:            c.Layout,
	}

	return timeParser, nil
}

// TimeParser is a plugin that parses time from an entry.
type TimeParser struct {
	helper.TransformerPlugin
	ParseFrom    entry.Field
	LayoutFlavor string
	Layout       string
}

// CanOutput will always return true for a parser plugin.
func (t *TimeParser) CanOutput() bool {
	return true
}

// Process will parse time from an entry.
func (t *TimeParser) Process(ctx context.Context, entry *entry.Entry) error {
	value, ok := entry.Get(t.ParseFrom)
	if !ok {
		return errors.NewError(
			"Log entry does not have the expected parse_from field.",
			"Ensure that all entries forwarded to this parser contain the parse_from field.",
			"parse_from", t.ParseFrom.String(),
		)
	}

	switch t.LayoutFlavor {
	case strptimeKey:
		timeValue, err := t.parseStrptime(value)
		if err != nil {
			return err
		}
		entry.Timestamp = timeValue
	case gotimeKey:
		timeValue, err := t.parseGotime(value)
		if err != nil {
			return err
		}
		entry.Timestamp = timeValue
	}

	return t.Output.Process(ctx, entry)
}

// Parse will parse a value as a time.
func (t *TimeParser) parseStrptime(value interface{}) (time.Time, error) {
	switch v := value.(type) {
	case string:
		return strptime.Parse(t.Layout, v)
	case []byte:
		return strptime.Parse(t.Layout, string(v))
	default:
		return time.Time{}, fmt.Errorf("type %T cannot be parsed as a time", value)
	}
}

// Parse will parse a value as a time.
func (t *TimeParser) parseGotime(value interface{}) (time.Time, error) {
	switch v := value.(type) {
	case string:
		return time.Parse(t.Layout, v)
	case []byte:
		return time.Parse(t.Layout, string(v))
	default:
		return time.Time{}, fmt.Errorf("type %T cannot be parsed as a time", value)
	}
}
