package builtin

import (
	"context"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/errors"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/helper"
	"go.uber.org/zap"
)

func init() {
	plugin.Register("metadata", &MetadataPluginConfig{})
}

type MetadataPluginConfig struct {
	helper.TransformerConfig `yaml:",inline"`

	Labels map[string]helper.ExprStringConfig `json:"labels" yaml:"labels"`
	Tags   []helper.ExprStringConfig          `json:"tags"   yaml:"tags"`
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
	labels map[string]*helper.ExprString
}

func (l *labeler) Label(e *entry.Entry) error {
	env := map[string]interface{}{
		"$": e.Record,
	}

	for k, v := range l.labels {
		rendered, err := v.Render(env)
		if err != nil {
			return err
		}
		e.Labels[k] = rendered
	}

	return nil
}

func buildLabeler(config map[string]helper.ExprStringConfig) (*labeler, error) {
	labels := make(map[string]*helper.ExprString)

	for k, v := range config {
		exprString, err := v.Build()
		if err != nil {
			return nil, err
		}

		labels[k] = exprString
	}

	return &labeler{labels}, nil
}

type tagger struct {
	tags []*helper.ExprString
}

func (t *tagger) Tag(e *entry.Entry) error {
	env := map[string]interface{}{
		"$": e.Record,
	}

	for _, v := range t.tags {
		rendered, err := v.Render(env)
		if err != nil {
			return err
		}
		e.Tags = append(e.Tags, rendered)
	}

	return nil
}

func buildTagger(config []helper.ExprStringConfig) (*tagger, error) {
	tags := make([]*helper.ExprString, 0, len(config))

	for _, v := range config {
		exprString, err := v.Build()
		if err != nil {
			return nil, err
		}

		tags = append(tags, exprString)
	}

	return &tagger{tags}, nil
}
