package builtin

import (
	"fmt"
	"reflect"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/helper"
	"github.com/mitchellh/mapstructure"
)

func init() {
	plugin.Register("restructure", &RestructurePluginConfig{}, OpDecoder)
}

type RestructurePluginConfig struct {
	helper.BasicPluginConfig      `mapstructure:",squash" yaml:",inline"`
	helper.BasicTransformerConfig `mapstructure:",squash" yaml:",inline"`

	Ops []Op `mapstructure:"ops" yaml:"ops"`
}

func (c RestructurePluginConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
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

		ops: c.Ops,
	}

	return plugin, nil
}

type RestructurePlugin struct {
	helper.BasicPlugin
	helper.BasicLifecycle
	helper.BasicTransformer

	ops []Op
}

func (p *RestructurePlugin) Process(e *entry.Entry) error {
	for _, op := range p.ops {
		op.Apply(e)
	}

	return p.Output.Process(e)
}

/*
 * Op Definitions
 */

type Op interface {
	Apply(entry *entry.Entry)
}

type OpRemove struct {
	Field entry.FieldSelector
}

func (op *OpRemove) Apply(e *entry.Entry) {
	e.Delete(op.Field)
}

type OpRetain struct {
	Fields []entry.FieldSelector
}

func (op *OpRetain) Apply(e *entry.Entry) {
	newEntry := entry.NewEntry()
	newEntry.Timestamp = e.Timestamp
	for _, field := range op.Fields {
		val, ok := e.Get(field)
		if !ok {
			continue
		}
		newEntry.Set(field, val)
	}
	*e = *newEntry
}

type OpMove struct {
	From entry.FieldSelector
	To   entry.FieldSelector
}

func (op *OpMove) Apply(e *entry.Entry) {
	val, ok := e.Delete(op.From)
	if !ok {
		return
	}

	e.Set(op.To, val)
}

type OpFlatten struct {
	Field entry.FieldSelector
}

func (op *OpFlatten) Apply(e *entry.Entry) {
	parent := op.Field.Parent()
	val, ok := e.Delete(op.Field)
	if !ok {
		// The field doesn't exist, so ignore it
		return
	}

	valMap, ok := val.(map[string]interface{})
	if !ok {
		// The field we were asked to flatten was not a map, so put it back
		e.Set(op.Field, val)
		return
	}

	for k, v := range valMap {
		e.Set(parent.Child(k), v)
	}
}

/*
 * Decoding
 */

var OpDecoder mapstructure.DecodeHookFunc = func(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
	if t.String() != "builtin.Op" {
		return data, nil
	}

	if f != reflect.TypeOf(map[interface{}]interface{}{}) {
		return nil, fmt.Errorf("cannot unmarshal a builtin.Op from type %s", f)
	}

	m := data.(map[interface{}]interface{})

	var opType *string
	var rawOp interface{}
	for k, v := range m {
		if opType != nil {
			return nil, fmt.Errorf("only one Op type can be defined per operation")
		}

		kStr, ok := k.(string)
		if !ok {
			return nil, fmt.Errorf("Op type must be a string")
		}

		opType = &kStr
		rawOp = v
	}

	if opType == nil {
		return nil, fmt.Errorf("no Op type defined")
	}

	switch *opType {
	case "move":
		var move OpMove
		err := decodeWithFieldSelector(rawOp, &move)
		if err != nil {
			return nil, fmt.Errorf("failed to decode OpMove: %s", err)
		}
		return &move, nil
	case "remove":
		var field entry.FieldSelector
		err := decodeWithFieldSelector(rawOp, &field)
		if err != nil {
			return nil, fmt.Errorf("failed to decode OpRemove: %s", err)
		}
		return &OpRemove{field}, nil
	case "retain":
		var fields []entry.FieldSelector
		err := decodeWithFieldSelector(rawOp, &fields)
		if err != nil {
			return nil, fmt.Errorf("failed to decode OpRetain: %s", err)
		}
		return &OpRetain{fields}, nil
	case "flatten":
		var field entry.FieldSelector
		err := decodeWithFieldSelector(rawOp, &field)
		if err != nil {
			return nil, fmt.Errorf("failed to decode Opflatten: %s", err)
		}
		return &OpFlatten{field}, nil
	default:
		return nil, fmt.Errorf("unknown Op type %s", *opType)
	}
}

func decodeWithFieldSelector(input, dest interface{}) error {
	cfg := &mapstructure.DecoderConfig{
		Result:     dest,
		DecodeHook: entry.FieldSelectorDecoder,
	}

	decoder, err := mapstructure.NewDecoder(cfg)
	if err != nil {
		return fmt.Errorf("build decoder: %s", err)
	}

	return decoder.Decode(input)
}
