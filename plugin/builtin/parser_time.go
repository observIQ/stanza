package builtin

import (
	"context"
	"fmt"
	"time"

	"github.com/pbnjay/strptime"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/errors"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/helper"
)

const strptimeKey = "strptime"
const gotimeKey = "gotime"

func init() {
	plugin.Register("time_parser", &TimeParserConfig{})
}

// TimeParserConfig is the configuration of a time parser plugin.
type TimeParserConfig struct {
	helper.TransformerConfig `yaml:",inline"`

	ParseFrom    entry.Field `json:"parse_from" yaml:"parse_from"`
	Layout       string      `json:"layout" yaml:"layout"`
	LayoutFlavor string      `json:"layout_flavor" yaml:"layout_flavor"` // strptime | gotime
	// TOOD OnError?
}

// Build will build a time parser plugin.
func (c TimeParserConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	transformerPlugin, err := c.TransformerConfig.Build(context)
	if err != nil {
		return nil, err
	}

	// if c.OnError == "" {
	// 	c.OnError = "ignore"
	// }

	// switch c.OnError {
	// case "fail", "drop", "ignore":
	// default:
	// 	return TimeParser{}, errors.NewError(
	// 		"Plugin config has an invalid `on_error` field.",
	// 		"Ensure that the `on_error` field is set to fail, drop, or ignore.",
	// 		"on_error", c.OnError,
	// 	)
	// }

	if c.LayoutFlavor == "" {
		c.LayoutFlavor = strptimeKey
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

// Process will parse time from an entry.
func (t *TimeParser) Process(ctx context.Context, entry *entry.Entry) error {
	switch t.LayoutFlavor {
	case strptimeKey:
		return t.TransformerPlugin.ProcessWith(ctx, entry, t.parseStrptime)
	case gotimeKey:
		return t.TransformerPlugin.ProcessWith(ctx, entry, t.parseGotime)
	default:
		return fmt.Errorf("unsupported layout_flavor %s", t.LayoutFlavor)
	}
}

// Parse will parse a value as a time.
func (t *TimeParser) parseStrptime(value interface{}) (interface{}, error) {
	switch v := value.(type) {
	case string:
		return strptime.Parse(v, t.Layout)
	case []byte:
		return strptime.Parse(string(v), t.Layout)
	default:
		return nil, fmt.Errorf("type %T cannot be parsed as a time", value)
	}
}

// Parse will parse a value as a time.
func (t *TimeParser) parseGotime(value interface{}) (interface{}, error) {
	switch v := value.(type) {
	case string:
		return time.Parse(t.Layout, v)
	case []byte:
		return time.Parse(t.Layout, string(v))
	default:
		return nil, fmt.Errorf("type %T cannot be parsed as a time", value)
	}
}
