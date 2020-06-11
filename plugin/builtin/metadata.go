package builtin

import (
	"context"
	"fmt"
	"reflect"
	"regexp"

	"github.com/antonmedv/expr"
	"github.com/antonmedv/expr/conf"
	"github.com/antonmedv/expr/vm"
	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/errors"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/helper"
	"go.uber.org/zap"
)

var stringLiteralPattern = regexp.MustCompile(`^[-_0-9A-z ]*$`)

func init() {
	plugin.Register("metadata", &MetadataPluginConfig{})
}

type MetadataPluginConfig struct {
	helper.TransformerConfig `yaml:",inline"`

	Labels map[string]string `json:"labels" yaml:"labels"`
	Tags   []string          `json:"tags"   yaml:"tags"`
}

func (c MetadataPluginConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	transformerPlugin, err := c.TransformerConfig.Build(context)
	if err != nil {
		return nil, err
	}

	labeler, err := buildLabeler(c.Labels)
	if err != nil {
		return nil, errors.Wrap(err, "validate labels")
	}

	tagger, err := buildTagger(c.Tags)
	if err != nil {
		return nil, errors.Wrap(err, "validate labels")
	}

	restructurePlugin := &MetadataPlugin{
		TransformerPlugin: transformerPlugin,
		labeler:           labeler,
		tagger:            tagger,
	}

	return restructurePlugin, nil
}

type MetadataPlugin struct {
	helper.TransformerPlugin
	labeler *labeler
	tagger  *tagger
}

func (p *MetadataPlugin) Process(ctx context.Context, e *entry.Entry) error {
	err := p.labeler.Label(e)
	if err != nil {
		p.Warnw("Failed to apply labels", zap.Error(err))
	}

	err = p.tagger.Tag(e)
	if err != nil {
		p.Warnw("Failed to apply tags", zap.Error(err))
	}

	return p.Output.Process(ctx, e)
}

type labeler struct {
	labels map[string]interface{}
}

func (l *labeler) Label(e *entry.Entry) error {
	for k, v := range l.labels {
		switch val := v.(type) {
		case *vm.Program:
			env := map[string]interface{}{
				"$": e.Record,
			}
			out, err := vm.Run(val, env)
			if err != nil {
				return err
			}
			if outString, ok := out.(string); ok {
				e.Labels[k] = outString
			} else {
				return fmt.Errorf("label expression generated non-string type %T", out)
			}
		case string:
			e.Labels[k] = val
		default:
			return fmt.Errorf("cannot create label with type %T", v)
		}
	}

	return nil
}

func buildLabeler(config map[string]string) (*labeler, error) {
	labels := make(map[string]interface{})

	for k, v := range config {
		if stringLiteralPattern.MatchString(v) {
			labels[k] = v
		} else {
			program, err := expr.Compile(v, expr.AllowUndefinedVariables(), asStringOption())
			if err != nil {
				return nil, errors.NewError("failed to compile expression",
					"ensure that your label only contains characters in [-_0-9A-z ], otherwise it will be interpreted as an expression",
					"error", err.Error(),
				)
			}
			labels[k] = program
		}
	}

	return &labeler{labels}, nil
}

type tagger struct {
	tags []interface{}
}

func (t *tagger) Tag(e *entry.Entry) error {
	for _, v := range t.tags {
		switch val := v.(type) {
		case *vm.Program:
			env := map[string]interface{}{
				"$": e.Record,
			}
			out, err := vm.Run(val, env)
			if err != nil {
				return err
			}
			if outString, ok := out.(string); ok {
				e.Tags = append(e.Tags, outString)
			} else {
				return fmt.Errorf("label expression generated non-string type %T", out)
			}
		case string:
			e.Tags = append(e.Tags, val)
		default:
			return fmt.Errorf("cannot create tag with type %T", v)
		}
	}

	return nil
}

func buildTagger(config []string) (*tagger, error) {
	tags := make([]interface{}, len(config))

	for i, v := range config {
		if stringLiteralPattern.MatchString(v) {
			tags[i] = v
		} else {
			program, err := expr.Compile(v, expr.AllowUndefinedVariables(), asStringOption())
			if err != nil {
				return nil, errors.NewError("failed to compile expression",
					"ensure that your label only contains characters in [-_0-9A-z ], otherwise it will be interpreted as an expression",
					"error", err.Error(),
				)
			}
			tags[i] = program
		}
	}

	return &tagger{tags}, nil
}

func asStringOption() expr.Option {
	return func(c *conf.Config) {
		c.Expect = reflect.String
	}
}
