package builtin

import (
	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/helper"
)

func init() {
	plugin.Register("restructure", &RestructureConfig{})
}

type RestructureConfig struct {
	helper.BasicPluginConfig      `mapstructure:",squash" yaml:",inline"`
	helper.BasicTransformerConfig `mapstructure:",squash" yaml:",inline"`

	Move   []MoveConfig          `mapstructure:"move" yaml:"move"`
	Remove []entry.FieldSelector `mapstructure:"remove" yaml:"remove"`
	Retain []entry.FieldSelector `mapstructure:"retain" yaml:"retain"`
}

type MoveConfig struct {
	From entry.FieldSelector `mapstructure:"from" yaml:"from"`
	To   entry.FieldSelector `mapstructure:"to" yaml:"to"`
}

// Build will build a JSON parser plugin.
func (c RestructureConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	basicPlugin, err := c.BasicPluginConfig.Build(context.Logger)
	if err != nil {
		return nil, err
	}

	basicTransformer, err := c.BasicTransformerConfig.Build()
	if err != nil {
		return nil, err
	}

	plugin := &RestructurePlugin{
		BasicPlugin:      basicPlugin,
		BasicTransformer: basicTransformer,

		move:   c.Move,
		remove: c.Remove,
		retain: c.Retain,
	}

	return plugin, nil
}

type RestructurePlugin struct {
	helper.BasicPlugin
	helper.BasicLifecycle
	helper.BasicTransformer

	move   []MoveConfig
	remove []entry.FieldSelector
	retain []entry.FieldSelector
}

// Process will parse an entry field as JSON.
func (p *RestructurePlugin) Process(e *entry.Entry) error {
	for _, moveConfig := range p.move {
		field, ok := e.Delete(moveConfig.From)
		if !ok {
			p.Debugw("Could not move field because it does not exist", "field", field)
			continue
		}

		e.Set(moveConfig.To, field)
	}

	for _, removeSelector := range p.remove {
		e.Delete(removeSelector)
	}

	if len(p.retain) > 0 {
		newEntry := entry.NewEntry()
		newEntry.Timestamp = e.Timestamp
		for _, retainSelector := range p.retain {
			field, ok := e.Get(retainSelector)
			if !ok {
				p.Warn("Could not retain field '%s' because it does not exist", retainSelector)
				continue
			}
			newEntry.Set(retainSelector, field)
		}
		return p.Output.Process(newEntry)
	}

	return p.Output.Process(e)
}
