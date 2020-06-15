package builtin

import (
	"context"
	"fmt"
	"strconv"
	"strings"
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
const epochKey = "epoch"

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
	case strptimeKey, gotimeKey, epochKey: // ok
	case "":
		c.LayoutFlavor = strptimeKey
	default:
		return nil, errors.NewError(
			fmt.Sprintf("unsupported layout_flavor %s", c.LayoutFlavor),
			"valid values are 'strptime', 'gotime', and 'epoch'",
			"plugin_id", c.PluginID,
			"plugin_type", c.PluginType,
		)
	}

	if c.Layout == "" {
		return nil, errors.NewError(
			"missing required configuration parameter `layout`",
			"specify 'strptime', 'gotime', or 'epoch'",
			"plugin_id", c.PluginID,
			"plugin_type", c.PluginType,
		)
	}

	if c.LayoutFlavor == strptimeKey {
		c.Layout, err = strptime.ToNative(c.Layout)
		if err != nil {
			return nil, errors.WithDetails(
				errors.Wrap(err, "parse strptime layout"),
				"plugin_id", c.PluginID,
				"plugin_type", c.PluginType,
			)
		}
		c.LayoutFlavor = gotimeKey
	} else if c.LayoutFlavor == epochKey {
		switch c.Layout {
		case "s", "ms", "us", "ns", "s.ms", "s.us", "s.ns": // ok
		default:
			return nil, errors.NewError(
				"invalid `layout` for `epoch` flavor",
				"specify 's', 'ms', 'us', 'ns', 's.ms', 's.us', or 's.ns'",
				"plugin_id", c.PluginID,
				"plugin_type", c.PluginType,
			)
		}
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
			"log entry does not have the expected parse_from field",
			"ensure that all entries forwarded to this parser contain the parse_from field",
			"parse_from", t.ParseFrom.String(),
		)
	}

	switch t.LayoutFlavor {
	case gotimeKey:
		timeValue, err := t.parseGotime(value)
		if err != nil {
			return err
		}
		entry.Timestamp = timeValue
	case epochKey:
		timeValue, err := t.parseEpochTime(value)
		if err != nil {
			return err
		}
		entry.Timestamp = timeValue
	default:
		return fmt.Errorf("unsupported layout flavor: %s", t.LayoutFlavor)
	}

	return t.Output.Process(ctx, entry)
}

func (t *TimeParser) parseGotime(value interface{}) (time.Time, error) {
	switch v := value.(type) {
	case string:
		return time.Parse(t.Layout, v)
	default:
		return time.Time{}, fmt.Errorf("type %T cannot be parsed as a time", value)
	}
}

func (t *TimeParser) parseEpochTime(value interface{}) (time.Time, error) {

	stamp, err := getEpochStamp(t.Layout, value)
	if err != nil {
		return time.Time{}, err
	}

	switch t.Layout {
	case "s", "ms", "us", "ns":
		i, err := strconv.ParseInt(stamp, 10, 64)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid value '%v' for layout '%s'", stamp, t.Layout)
		}
		return toTime[t.Layout](i), nil
	case "s.ms", "s.us", "s.ns":
		secSubsec := strings.Split(stamp, ".")
		if len(secSubsec) != 2 {
			return time.Time{}, fmt.Errorf("invalid value '%v' for layout '%s'", stamp, t.Layout)
		}
		sec, secErr := strconv.ParseInt(secSubsec[0], 10, 64)
		subsec, subsecErr := strconv.ParseInt(secSubsec[1], 10, 64)
		if secErr != nil || subsecErr != nil {
			return time.Time{}, fmt.Errorf("invalid value '%v' for layout '%s'", stamp, t.Layout)
		}
		return time.Unix(sec, subsec*subsecToNs[t.Layout]), nil
	default:
		return time.Time{}, fmt.Errorf("invalid layout '%s'", t.Layout)
	}
}

func getEpochStamp(layout string, value interface{}) (string, error) {
	switch v := value.(type) {
	case string:
		return v, nil
	case int, int32, int64, uint32, uint64:
		switch layout {
		case "s", "ms", "us", "ns":
			return fmt.Sprintf("%d", v), nil
		case "s.ms", "s.us", "s.ns":
			return fmt.Sprintf("%10.9f", v), nil
		default:
			return "", fmt.Errorf("invalid layout '%s'", layout)
		}
	default:
		return "", fmt.Errorf("type %T cannot be parsed as a time", v)
	}
}

type toTimeFunc = func(int64) time.Time

var toTime = map[string]toTimeFunc{
	"s":  func(s int64) time.Time { return time.Unix(s/1, 0) },
	"ms": func(ms int64) time.Time { return time.Unix(ms/1e3, (ms%1e3)*1e6) },
	"us": func(us int64) time.Time { return time.Unix(us/1e6, (us%1e6)*1e3) },
	"ns": func(ns int64) time.Time { return time.Unix(0, ns) },
}
var subsecToNs = map[string]int64{"s.ms": 1e6, "s.us": 1e3, "s.ns": 1}
