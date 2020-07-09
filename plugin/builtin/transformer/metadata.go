package transformer

import (
	"context"

	"github.com/observiq/carbon/entry"
	"github.com/observiq/carbon/errors"
	"github.com/observiq/carbon/plugin"
	"github.com/observiq/carbon/plugin/helper"
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

// Process will process an incoming entry using the metadata transform.
func (p *MetadataPlugin) Process(ctx context.Context, entry *entry.Entry) error {
	return p.ProcessWith(ctx, entry, p.Transform)
}

// Transform will transform an entry using the labeler and tagger.
func (p *MetadataPlugin) Transform(entry *entry.Entry) (*entry.Entry, error) {
	err := p.labeler.Label(entry)
	if err != nil {
		return entry, err
	}

	err = p.tagger.Tag(entry)
	if err != nil {
		return entry, err
	}

	return entry, nil
}

type labeler struct {
	labels map[string]*helper.ExprString
}

func (l *labeler) Label(e *entry.Entry) error {
	env := helper.GetExprEnv(e)
	defer helper.PutExprEnv(env)

	for k, v := range l.labels {
		rendered, err := v.Render(env)
		if err != nil {
			return err
		}
		e.AddLabel(k, rendered)
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
	env := helper.GetExprEnv(e)
	defer helper.PutExprEnv(env)

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
