package builtin

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/antonmedv/expr"
	"github.com/antonmedv/expr/vm"
	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/helper"
	"go.uber.org/zap"
)

func init() {
	plugin.Register("restructure", &RestructurePluginConfig{})
}

type RestructurePluginConfig struct {
	helper.TransformerConfig `yaml:",inline"`

	Ops []Op `json:"ops" yaml:"ops"`
}

func (c RestructurePluginConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	transformerPlugin, err := c.TransformerConfig.Build(context)
	if err != nil {
		return nil, err
	}

	restructurePlugin := &RestructurePlugin{
		TransformerPlugin: transformerPlugin,
		ops:               c.Ops,
	}

	return restructurePlugin, nil
}

type RestructurePlugin struct {
	helper.TransformerPlugin
	ops []Op
}

func (p *RestructurePlugin) Process(ctx context.Context, e *entry.Entry) error {
	for _, op := range p.ops {
		err := op.Apply(e)
		if err != nil {
			p.Warnw("Failed to apply operation", zap.Error(err), "entry", e)
		}
	}

	return p.Output.Process(ctx, e)
}

/*****************
  Op Definitions
*****************/

type Op struct {
	OpApplier
}

type OpApplier interface {
	Apply(entry *entry.Entry) error
	Type() string
}

func (o *Op) UnmarshalJSON(raw []byte) error {
	var typeDecoder map[string]rawMessage
	err := json.Unmarshal(raw, &typeDecoder)
	if err != nil {
		return err
	}

	return o.unmarshalDecodedType(typeDecoder)
}

func (o *Op) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var typeDecoder map[string]rawMessage
	err := unmarshal(&typeDecoder)
	if err != nil {
		return err
	}

	return o.unmarshalDecodedType(typeDecoder)
}

type rawMessage struct {
	unmarshal func(interface{}) error
}

func (msg *rawMessage) UnmarshalYAML(unmarshal func(interface{}) error) error {
	msg.unmarshal = unmarshal
	return nil
}

func (msg *rawMessage) UnmarshalJSON(raw []byte) error {
	msg.unmarshal = func(dest interface{}) error {
		return json.Unmarshal(raw, dest)
	}
	return nil
}

func (msg *rawMessage) Unmarshal(v interface{}) error {
	return msg.unmarshal(v)
}

func (o *Op) unmarshalDecodedType(typeDecoder map[string]rawMessage) error {
	var rawMessage rawMessage
	var opType string
	for k, v := range typeDecoder {
		if opType != "" {
			return fmt.Errorf("only one Op type can be defined per operation")
		}
		opType = k
		rawMessage = v
	}

	if opType == "" {
		return fmt.Errorf("no Op type defined")
	}

	if rawMessage.unmarshal == nil {
		return fmt.Errorf("op fields cannot be empty")
	}

	var err error
	switch opType {
	case "move":
		var move OpMove
		err = rawMessage.Unmarshal(&move)
		if err != nil {
			return err
		}
		o.OpApplier = &move
	case "add":
		var add OpAdd
		err = rawMessage.Unmarshal(&add)
		if err != nil {
			return err
		}
		o.OpApplier = &add
	case "remove":
		var remove OpRemove
		err = rawMessage.Unmarshal(&remove)
		if err != nil {
			return err
		}
		o.OpApplier = &remove
	case "retain":
		var retain OpRetain
		err = rawMessage.Unmarshal(&retain)
		if err != nil {
			return err
		}
		o.OpApplier = &retain
	case "flatten":
		var flatten OpFlatten
		err = rawMessage.Unmarshal(&flatten)
		if err != nil {
			return err
		}
		o.OpApplier = &flatten
	default:
		return fmt.Errorf("unknown op type '%s'", opType)
	}

	return nil
}

func (o Op) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		o.Type(): o.OpApplier,
	})
}

func (o Op) MarshalYAML() (interface{}, error) {
	return map[string]interface{}{
		o.Type(): o.OpApplier,
	}, nil
}

/******
  Add
******/

type OpAdd struct {
	Field     entry.Field `json:"field" yaml:"field"`
	Value     interface{} `json:"value,omitempty" yaml:"value,omitempty"`
	program   *vm.Program
	ValueExpr *string `json:"value_expr,omitempty" yaml:"value_expr,omitempty"`
}

func (op *OpAdd) Apply(e *entry.Entry) error {
	switch {
	case op.Value != nil:
		e.Set(op.Field, op.Value)
	case op.program != nil:
		env := map[string]interface{}{
			"$": e.Record,
		}
		result, err := vm.Run(op.program, env)
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

func (op *OpAdd) Type() string {
	return "add"
}

type opAddRaw struct {
	Field     *entry.Field `json:"field"      yaml:"field"`
	Value     interface{}  `json:"value"      yaml:"value"`
	ValueExpr *string      `json:"value_expr" yaml:"value_expr"`
}

func (op *OpAdd) UnmarshalJSON(raw []byte) error {
	var addRaw opAddRaw
	err := json.Unmarshal(raw, &addRaw)
	if err != nil {
		return fmt.Errorf("decode OpAdd: %s", err)
	}

	return op.unmarshalFromOpAddRaw(addRaw)
}

func (op *OpAdd) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var addRaw opAddRaw
	err := unmarshal(&addRaw)
	if err != nil {
		return fmt.Errorf("decode OpAdd: %s", err)
	}

	return op.unmarshalFromOpAddRaw(addRaw)
}

func (op *OpAdd) unmarshalFromOpAddRaw(addRaw opAddRaw) error {
	if addRaw.Field == nil {
		return fmt.Errorf("decode OpAdd: missing required field 'field'")
	}

	switch {
	case addRaw.Value != nil && addRaw.ValueExpr != nil:
		return fmt.Errorf("decode OpAdd: only one of 'value' or 'value_expr' may be defined")
	case addRaw.Value == nil && addRaw.ValueExpr == nil:
		return fmt.Errorf("decode OpAdd: exactly one of 'value' or 'value_expr' must be defined")
	case addRaw.Value != nil:
		op.Field = *addRaw.Field
		op.Value = addRaw.Value
	case addRaw.ValueExpr != nil:
		compiled, err := expr.Compile(*addRaw.ValueExpr, expr.AllowUndefinedVariables())
		if err != nil {
			return fmt.Errorf("decode OpAdd: failed to compile expression '%s': %w", *addRaw.ValueExpr, err)
		}
		op.Field = *addRaw.Field
		op.program = compiled
		op.ValueExpr = addRaw.ValueExpr
	}

	return nil
}

/*********
  Remove
*********/

type OpRemove struct {
	Field entry.Field
}

func (op *OpRemove) Apply(e *entry.Entry) error {
	e.Delete(op.Field)
	return nil
}

func (op *OpRemove) Type() string {
	return "remove"
}

func (op *OpRemove) UnmarshalJSON(raw []byte) error {
	return json.Unmarshal(raw, &op.Field)
}

func (op *OpRemove) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return unmarshal(&op.Field)
}

func (op OpRemove) MarshalJSON() ([]byte, error) {
	return json.Marshal(op.Field)
}

func (op OpRemove) MarshalYAML() (interface{}, error) {
	return op.Field.String(), nil
}

/*********
  Retain
*********/

type OpRetain struct {
	Fields []entry.Field
}

func (op *OpRetain) Apply(e *entry.Entry) error {
	newEntry := entry.New()
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

func (op *OpRetain) Type() string {
	return "retain"
}

func (op *OpRetain) UnmarshalJSON(raw []byte) error {
	return json.Unmarshal(raw, &op.Fields)
}

func (op *OpRetain) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return unmarshal(&op.Fields)
}

func (op OpRetain) MarshalJSON() ([]byte, error) {
	return json.Marshal(op.Fields)
}

func (op OpRetain) MarshalYAML() (interface{}, error) {
	return op.Fields, nil
}

/*******
  Move
*******/

type OpMove struct {
	From entry.Field `json:"from" yaml:"from,flow"`
	To   entry.Field `json:"to" yaml:"to,flow"`
}

func (op *OpMove) Apply(e *entry.Entry) error {
	val, ok := e.Delete(op.From)
	if !ok {
		return fmt.Errorf("apply move: field %s does not exist on record", op.From)
	}

	e.Set(op.To, val)
	return nil
}

func (op *OpMove) Type() string {
	return "move"
}

/**********
  Flatten
**********/

type OpFlatten struct {
	Field entry.Field
}

func (op *OpFlatten) Apply(e *entry.Entry) error {
	fs := entry.Field(op.Field)
	parent := fs.Parent()
	val, ok := e.Delete(fs)
	if !ok {
		// The field doesn't exist, so ignore it
		return fmt.Errorf("apply flatten: field %s does not exist on record", fs)
	}

	valMap, ok := val.(map[string]interface{})
	if !ok {
		// The field we were asked to flatten was not a map, so put it back
		e.Set(fs, val)
		return fmt.Errorf("apply flatten: field %s is not a map", fs)
	}

	for k, v := range valMap {
		e.Set(parent.Child(k), v)
	}
	return nil
}

func (op *OpFlatten) Type() string {
	return "flatten"
}

func (op *OpFlatten) UnmarshalJSON(raw []byte) error {
	return json.Unmarshal(raw, &op.Field)
}

func (op *OpFlatten) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return unmarshal(&op.Field)
}

func (op OpFlatten) MarshalJSON() ([]byte, error) {
	return json.Marshal(op.Field)
}

func (op OpFlatten) MarshalYAML() (interface{}, error) {
	return op.Field.String(), nil
}
