package helper

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
)

// StrptimeKey is literally "strptime", and is the default flavor
const StrptimeKey = "strptime"

// GotimeKey is literally "gotime" and uses Golang's native time.Parse
const GotimeKey = "gotime"

// EpochKey is literally "epoch" and can parse seconds and/or subseconds
const EpochKey = "epoch"

// NativeKey is literally "native" and refers to Golang's native time.Time
const NativeKey = "native" // provided for plugin development

// TimeParser is a helper that parses time onto an entry.
type TimeParser struct {
	ParseFrom    entry.Field `json:"parse_from,omitempty" yaml:"parse_from,omitempty"`
	Layout       string      `json:"layout,omitempty" yaml:"layout,omitempty"`
	LayoutFlavor string      `json:"layout_flavor,omitempty" yaml:"layout_flavor,omitempty"`
	Preserve     bool        `json:"preserve"   yaml:"preserve"`
}

// IsZero returns true if the TimeParser is not a valid config
func (t *TimeParser) IsZero() bool {
	return t.Layout == ""
}

// Validate validates a TimeParser, and reconfigures it if necessary
func (t *TimeParser) Validate(context plugin.BuildContext) error {

	if t.Layout == "" && t.LayoutFlavor != "native" {
		return errors.NewError(
			"missing required configuration parameter `layout`",
			"specify 'strptime', 'gotime', or 'epoch'",
		)
	}

	if t.LayoutFlavor == "" {
		t.LayoutFlavor = StrptimeKey
	}

	switch t.LayoutFlavor {
	case NativeKey, GotimeKey: // ok
	case StrptimeKey:
		var err error
		t.Layout, err = strptime.ToNative(t.Layout)
		if err != nil {
			return errors.WithDetails(
				errors.Wrap(err, "parse strptime layout"),
			)
		}
		t.LayoutFlavor = GotimeKey
	case EpochKey:
		switch t.Layout {
		case "s", "ms", "us", "ns", "s.ms", "s.us", "s.ns": // ok
		default:
			return errors.NewError(
				"invalid `layout` for `epoch` flavor",
				"specify 's', 'ms', 'us', 'ns', 's.ms', 's.us', or 's.ns'",
			)
		}
	default:
		return errors.NewError(
			fmt.Sprintf("unsupported layout_flavor %s", t.LayoutFlavor),
			"valid values are 'strptime', 'gotime', and 'epoch'",
		)
	}

	return nil
}

// Parse will parse time from a field and attach it to the entry
func (t *TimeParser) Parse(ctx context.Context, entry *entry.Entry) error {
	value, ok := entry.Get(t.ParseFrom)
	if !ok {
		return errors.NewError(
			"log entry does not have the expected parse_from field",
			"ensure that all entries forwarded to this parser contain the parse_from field",
			"parse_from", t.ParseFrom.String(),
		)
	}

	switch t.LayoutFlavor {
	case NativeKey:
		timeValue, ok := value.(time.Time)
		if !ok {
			return fmt.Errorf("native time.Time field required, but found: %v", value)
		}
		entry.Timestamp = timeValue
	case GotimeKey:
		timeValue, err := t.parseGotime(value)
		if err != nil {
			return err
		}
		entry.Timestamp = timeValue
	case EpochKey:
		timeValue, err := t.parseEpochTime(value)
		if err != nil {
			return err
		}
		entry.Timestamp = timeValue
	default:
		return fmt.Errorf("unsupported layout flavor: %s", t.LayoutFlavor)
	}

	if !t.Preserve {
		entry.Delete(t.ParseFrom)
	}

	return nil
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
	"s":  func(s int64) time.Time { return time.Unix(s, 0) },
	"ms": func(ms int64) time.Time { return time.Unix(ms/1e3, (ms%1e3)*1e6) },
	"us": func(us int64) time.Time { return time.Unix(us/1e6, (us%1e6)*1e3) },
	"ns": func(ns int64) time.Time { return time.Unix(0, ns) },
}
var subsecToNs = map[string]int64{"s.ms": 1e6, "s.us": 1e3, "s.ns": 1}
