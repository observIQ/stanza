package builtin

import (
	"fmt"
	"reflect"

	"github.com/antonmedv/expr"
	"github.com/antonmedv/expr/vm"
	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/helper"
	"github.com/mitchellh/mapstructure"
	"go.uber.org/zap"
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
		err := op.Apply(e)
		if err != nil {
			p.Warnw("Failed to apply operation", zap.Error(err), "entry", e)
		}
	}

	return p.Output.Process(e)
}

/*
 * Op Definitions
 */

type Op interface {
	Apply(entry *entry.Entry) error
}

type OpAdd struct {
	Field     entry.FieldSelector
	Value     interface{}
	ValueExpr *vm.Program
}

func (op *OpAdd) Apply(e *entry.Entry) error {
	switch {
	case op.Value != nil:
		e.Set(op.Field, op.Value)
	case op.ValueExpr != nil:
		env := map[string]interface{}{
			"record": e.Record,
		}
		result, err := vm.Run(op.ValueExpr, env)
		if err != nil {
			return fmt.Errorf("evaluate value_expr: %s", err)
		}
		e.Set(op.Field, result)
	default:
		// Should never reach here if we went through the unmarshalling code
		return fmt.Errorf("neither value or value_expr are are set")
	}

	return nil
}

type OpRemove struct {
	Field entry.FieldSelector
}

func (op *OpRemove) Apply(e *entry.Entry) error {
	e.Delete(op.Field)
	return nil
}

type OpRetain struct {
	Fields []entry.FieldSelector
}

func (op *OpRetain) Apply(e *entry.Entry) error {
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
	return nil
}

type OpMove struct {
	From entry.FieldSelector
	To   entry.FieldSelector
}

func (op *OpMove) Apply(e *entry.Entry) error {
	val, ok := e.Delete(op.From)
	if !ok {
		return fmt.Errorf("apply move: field %s does not exist on record", op.From)
	}

	e.Set(op.To, val)
	return nil
}

type OpFlatten struct {
	Field entry.FieldSelector
}

func (op *OpFlatten) Apply(e *entry.Entry) error {
	parent := op.Field.Parent()
	val, ok := e.Delete(op.Field)
	if !ok {
		// The field doesn't exist, so ignore it
		return fmt.Errorf("apply flatten: field %s does not exist on record", op.Field)
	}

	valMap, ok := val.(map[string]interface{})
	if !ok {
		// The field we were asked to flatten was not a map, so put it back
		e.Set(op.Field, val)
		return fmt.Errorf("apply flatten: field %s is not a map", op.Field)
	}

	for k, v := range valMap {
		e.Set(parent.Child(k), v)
	}
	return nil
}

/*
 * Decoding
 */

var OpDecoder mapstructure.DecodeHookFunc = func(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
	if t.String() != "builtin.Op" {
		return data, nil
	}

	var m map[string]interface{}
	switch f {
	case reflect.TypeOf(map[interface{}]interface{}{}):
		m = make(map[string]interface{})
		for k, v := range data.(map[interface{}]interface{}) {
			if kString, ok := k.(string); ok {
				m[kString] = v
			} else {
				return nil, fmt.Errorf("map has non-string key %v of type %T", k, k)
			}
		}
	case reflect.TypeOf(map[string]interface{}{}):
		m = data.(map[string]interface{})
	default:
		return data, nil
	}

	var opType *string
	var rawOp interface{}
	for k, v := range m {
		if opType != nil {
			return nil, fmt.Errorf("only one Op type can be defined per operation")
		}

		opType = &k
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
	case "add":
		var addRaw struct {
			Field     entry.FieldSelector
			Value     interface{}
			ValueExpr *string `mapstructure:"value_expr"`
		}
		err := decodeWithFieldSelector(rawOp, &addRaw)
		if err != nil {
			return nil, fmt.Errorf("failed to decode OpAdd: %s", err)
		}
		// TODO if add.Value is a map[interface{}]interface{}, convert it to map[string]interface{}

		if addRaw.Field == nil {
			return nil, fmt.Errorf("decode OpAdd: missing required field 'field'")
		}

		switch {
		case addRaw.Value != nil && addRaw.ValueExpr != nil:
			return nil, fmt.Errorf("decode OpAdd: only one of 'value' or 'value_expr' may be defined")
		case addRaw.Value == nil && addRaw.ValueExpr == nil:
			return nil, fmt.Errorf("decode OpAdd: exactly one of 'value' or 'value_expr' must be defined")
		case addRaw.Value != nil:
			return &OpAdd{
				Field: addRaw.Field,
				Value: addRaw.Value,
			}, nil
		case addRaw.ValueExpr != nil:
			compiled, err := expr.Compile(*addRaw.ValueExpr, expr.AllowUndefinedVariables())
			if err != nil {
				return nil, fmt.Errorf("decode OpAdd: failed to compile expression '%s': %w", *addRaw.ValueExpr, err)
			}
			return &OpAdd{
				Field:     addRaw.Field,
				ValueExpr: compiled,
			}, nil
		}
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
	}

	return nil, fmt.Errorf("unknown Op type %s", *opType)
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
