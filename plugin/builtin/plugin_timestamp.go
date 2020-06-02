package builtin

import (
	"context"
	"fmt"
	"time"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/helper"
)

func init() {
	plugin.Register("timestamp", &TimestampConfig{})
}

// TimestampConfig is the configuration of a timestamp plugin.
type TimestampConfig struct {
	helper.TransformerConfig `yaml:",inline"`

	CopyFrom    entry.Field `json:"copy_from,omitempty"    yaml:"copy_from,omitempty"`
	RemoveField bool        `json:"remove_field,omitempty" yaml:"remove_field,omitempty"`
}

// Build will build a timestamp plugin.
func (c TimestampConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	transformerPlugin, err := c.TransformerConfig.Build(context)
	if err != nil {
		return nil, err
	}

	timestampPlugin := &TimestampPlugin{
		TransformerPlugin: transformerPlugin,
		CopyFrom:          c.CopyFrom,
		RemoveField:       c.RemoveField,
	}

	return timestampPlugin, nil
}

// TimestampPlugin is a plugin that changes the timestamp of an entry.
type TimestampPlugin struct {
	helper.TransformerPlugin
	CopyFrom    entry.Field
	RemoveField bool
}

// Process will wait until a rate is met before sending an entry to the output.
func (t *TimestampPlugin) Process(ctx context.Context, entry *entry.Entry) error {
	value, ok := entry.Get(t.CopyFrom)
	if !ok {
		return fmt.Errorf("copy_from field '%s' does not exist on the record", t.CopyFrom)
	}

	switch v := value.(type) {
	case time.Time:
		entry.Timestamp = v
	default:
		return fmt.Errorf("Type '%T' cannot be converted to type 'Time'", value)
	}

	if t.RemoveField {
		entry.Delete(t.CopyFrom)
	}

	return t.Output.Process(ctx, entry)
}
